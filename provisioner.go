package main

import (
	"context"
	"fmt"

	client "github.com/finarfin/go-salt-client/cherrypy"
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
			"eauth": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cmd": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client": {
							Type:     schema.TypeString,
							Required: true,
						},
						"tgt": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"fun": {
							Type:     schema.TypeString,
							Required: true,
						},
						"arg": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional: true,
						},
						"kwarg": {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional: true,
						},
					},
				},
			},
		},

		ApplyFunc:    apply,
		ValidateFunc: validate,
	}
}

func apply(ctx context.Context) error {
	//connState := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	cli, err := client.NewClient(
		data.Get("address").(string),
		data.Get("username").(string),
		data.Get("password").(string),
		data.Get("eauth").(string),
	)

	if err != nil {
		return err
	}

	rawCmds := data.Get("cmd").(*schema.Set).List()
	cmds := make([]client.Command, len(rawCmds))
	for i, cmd := range rawCmds {
		rawCmd := cmd.(map[string]interface{})
		cmds[i] = client.Command{
			Client:   rawCmd["client"].(string),
			Target:   rawCmd["tgt"].(string),
			Function: rawCmd["fun"].(string),
		}

		if v, ok := rawCmd["arg"].([]interface{}); ok {
			cmds[i].Args = make([]string, len(v))
			for j, arg := range v {
				cmds[i].Args[j] = arg.(string)
			}
		}

		if v, ok := rawCmd["kwarg"].(map[string]interface{}); ok {
			cmds[i].Kwargs = make(map[string]string, len(v))
			for j, arg := range v {
				cmds[i].Kwargs[j] = arg.(string)
			}
		}
	}

	resp, err := cli.Run(cmds)
	if err != nil {
		return err
	}

	result := resp["return"].([]interface{})
	if len(result) != 1 {
		return fmt.Errorf("Test failed")
	}

	for _, res := range result {
		r := res.(map[string]interface{})
		if r["result"] == nil || !r["result"].(bool) {
			return fmt.Errorf("Failed here too")
		}
	}

	return nil
}

func validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	return ws, es
}
