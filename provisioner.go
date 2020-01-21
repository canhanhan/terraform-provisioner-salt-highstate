package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	salt "github.com/finarfin/go-salt-netapi-client/cherrypy"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:     schema.TypeString,
				Required: true,
			},
			"username": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
			},
			"backend": {
				Type:     schema.TypeString,
				Required: true,
			},
			"minion_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"timeout_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
			},
			"interval_secs": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  10,
			},
		},

		ApplyFunc:    apply,
		ValidateFunc: validate,
	}
}

func apply(ctx context.Context) error {
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	cli := salt.NewClient(
		d.Get("address").(string),
		d.Get("username").(string),
		d.Get("password").(string),
		d.Get("backend").(string),
	)

	if err := cli.Login(); err != nil {
		return err
	}

	timeout := time.Duration(d.Get("timeout_minutes").(int)) * time.Minute
	interval := time.Duration(d.Get("interval_secs").(int)) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	minion := d.Get("minion_id").(string)
	o.Output(fmt.Sprintf("Waiting for minion %s to register with master", minion))
	if err := waitForMinion(ctx, o, cli, minion, interval, timeout); err != nil {
		return err
	}

	o.Output(fmt.Sprintf("Executing state.highstate on minion %s", minion))
	result, err := cli.SubmitJob(salt.MinionJob{
		Target:     minion,
		TargetType: salt.List,
		Function:   "state.highstate",
	})
	if err != nil {
		return err
	}

	for {
		time.Sleep(interval)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, err := cli.RunJob(salt.Command{
				Client:   "runner",
				Function: "jobs.lookup_jid",
				Args:     []string{result.JobID},
			})
			if err != nil {
				return err
			}

			minions, ok := data["data"].(map[string]interface{})
			if !ok {
				o.Output("Job is not started yet")
				continue
			}

			if minionData, ok := minions[minion]; ok {
				lowstates, ok := minionData.(map[string]interface{})
				if !ok {
					log.Println("[TRACE] Highstate job is still executing for minion: " + minion)
					continue
				}

				for k, v := range lowstates {
					state := v.(map[string]interface{})
					if !state["result"].(bool) {
						return fmt.Errorf("State %s failed on %s: %s", k, minion, state["comment"])
					}
				}

				return nil
			}
		}

		log.Println("[TRACE] Minion has not yet reported status for highstate job: " + minion)
	}
}

func validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	return ws, es
}

func waitForMinion(ctx context.Context, o terraform.UIOutput, cli *salt.Client, minion string, interval time.Duration, timeout time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			minionData, err := cli.Minion(minion)
			if err != nil && !errors.Is(err, salt.ErrorMinionNotFound) {
				return err
			} else if minionData == nil || minionData.Grains == nil {
				time.Sleep(interval)
				continue
			}
		}

		break
	}

	return nil
}
