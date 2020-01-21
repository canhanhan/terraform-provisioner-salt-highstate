package main

import (
	"context"
	"errors"
	"fmt"
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
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	cli := salt.NewClient(
		data.Get("address").(string),
		data.Get("username").(string),
		data.Get("password").(string),
		data.Get("backend").(string),
	)

	if err := cli.Login(); err != nil {
		return err
	}

	timeout := time.Duration(data.Get("timeout_minutes").(int)) * time.Minute
	interval := time.Duration(data.Get("interval_secs").(int)) * time.Second
	minion := data.Get("minion_id").(string)
	o.Output(fmt.Sprintf("Waiting for minion %s to register with master", minion))
	if err := waitForMinion(ctx, o, cli, minion, interval, timeout); err != nil {
		return err
	}

	cmd := salt.Command{
		Client:     "local",
		Target:     data.Get("minion_id").(string),
		TargetType: salt.List,
		Function:   "state.highstate",
	}

	o.Output(fmt.Sprintf("Executing state.highstate on minion %s", minion))
	minions, err := cli.RunJob(cmd)
	if err != nil {
		return err
	}

	if len(minions) != 1 {
		return fmt.Errorf("Expected results from 1 minion but received %d", len(minions))
	}

	lowStates := minions[minion].(map[string]interface{})
	for k, v := range lowStates {
		state := v.(map[string]interface{})
		if state["result"] == nil || !state["result"].(bool) {
			return fmt.Errorf("State %s failed on %s: %s", k, minion, state["comment"])
		}
	}

	return nil
}

func validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	return ws, es
}

func waitForMinion(ctx context.Context, o terraform.UIOutput, cli *salt.Client, minion string, interval time.Duration, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
