package cplatform

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	confluent "github.com/wayarmy/gonfluent"
)

var (
	validRole = []string{
		"AuditAdmin",
		"ClusterAdmin",
		"DeveloperManage",
		"DeveloperRead",
		"DeveloperWrite",
		"Operator",
		"ResourceOwner",
		"SecurityAdmin",
		"SystemAdmin",
		"UserAdmin",
	}
	scopeRole = []string{
		"DeveloperRead",
		"DeveloperWrite",
		"Operator",
		"ResourceOwner",
	}
	validCluster = []string{
		"Kafka",
		"SchemaRegistry",
		"Connect",
		"KSQL",
	}
	validPatternType = []string{
		"LITERAL",
		"PREFIXED",
	}
)

// roleBindings will bind the roles to users/scope
// example:
// resource "cluster_role_binding" "user_confluent_test" {
//   principal  = "User:confluent-test"
//   role 		= "Operation"
//	 cluster_id = "zxvdlaskdjal"
// 	 cluster_type = "Kafka"
//
//   provider = confluent-kafka.confluent
//	}
// Resource ID = cluster_type + "|" + clusterId + "|" + principal + "|" + role
func clusterRoleBindings() *schema.Resource {
	return &schema.Resource{
		CreateContext: clusterRoleBindingsCreate,
		DeleteContext: clusterRoleBindingsDelete,
		ReadContext:   clusterRoleBindingsRead,

		Schema: map[string]*schema.Schema{
			"principal": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Enable communication with the Kafka Cluster over TLS.",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if !strings.Contains(v, "User:") || !strings.Contains(v, "Group:") && strings.Contains(v, "|") {
						errs = append(errs, fmt.Errorf("%q must be defined with USER: or GROUP: and must not have |, got: %s", key, v))
					}
					return
				},
			},
			"role": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				Description:  "Role of",
				DefaultFunc:  schema.EnvDefaultFunc("ROLE", "DeveloperRead"),
				ValidateFunc: validation.StringInSlice(validRole, false),
			},
			"cluster_type": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringInSlice(validCluster, false),
			},
			"cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The ID of cluster",
			},
		},
	}
}

func clusterRoleBindingsRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)

	clusterId := parseIdToResourcesList(d.Id())[1]
	principal := parseIdToResourcesList(d.Id())[2]
	role := parseIdToResourcesList(d.Id())[3]

	cDetails := &confluent.ClusterDetails{}
	var err error
	switch clusterType := parseIdToResourcesList(d.Id())[0]; clusterType {
	case "Kafka":
		cDetails.Clusters.KafkaCluster = clusterId
		_, err := c.LookupRoleBinding(principal, role, *cDetails)
		if err == nil {
			err = d.Set("role", role)
		}
		if err == nil {
			err = d.Set("principal", principal)
		}
		if err == nil {
			err = d.Set("cluster_id", clusterId)
		}
	case "SchemaRegistry":

	case "Connect":

	case "KSQL":
	}

	return diag.FromErr(err)
}

func clusterRoleBindingsCreate(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	clusterId := d.Get("cluster_id").(string)
	principal := d.Get("principal").(string)
	role := d.Get("role").(string)
	clusterType := d.Get("cluster_type").(string)

	switch clusterType {
	case "Kafka":
		err := bindKafkaClusterRoleBinding(c, clusterId, principal, role)
		if err != nil {
			return diag.FromErr(err)
		}
	case "SchemaRegistry":

	case "Connect":

	case "KSQL":
	}
	d.SetId(clusterType + "|" + clusterId + "|" + principal + "|" + role)
	return nil
}

func clusterRoleBindingsDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)

	clusterId := parseIdToResourcesList(d.Id())[1]
	principal := parseIdToResourcesList(d.Id())[2]
	role := parseIdToResourcesList(d.Id())[3]

	cDetails := &confluent.ClusterDetails{}

	switch clusterType := parseIdToResourcesList(d.Id())[0]; clusterType {
	case "Kafka":
		cDetails.Clusters.KafkaCluster = clusterId
		err := c.DeleteRoleBinding(principal, role, *cDetails)
		if err != nil {
			diag.FromErr(err)
		}
	case "SchemaRegistry":

	case "Connect":

	case "KSQL":
	}

	return nil
}

func parseIdToResourcesList(bindId string) []string {
	return strings.Split(bindId, "|")
}

func bindKafkaClusterRoleBinding(c *confluent.Client, clusterId, principal, role string) error {
	cDetails := &confluent.ClusterDetails{}
	cDetails.Clusters.KafkaCluster = clusterId

	err := c.BindPrincipalToRole(principal, role, *cDetails)
	if err != nil {
		return err
	}
	return nil
}
