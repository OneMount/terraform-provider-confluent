package cplatform

import (
	"context"
	"fmt"
	confluent "github.com/OneMount/gonfluent"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	"strings"
)

// schemaRegistryRBAC define the roles binding for schema registry resources
// example:
/*
resource "connectors_rbac" "example_role_binding_developerwrite_connector" {
	cluster_id = "5-2KZs0YRYCVv8YCDdfSVw"
	connect_cluster_id = "connect-taiwan"
	role = "DeveloperRead"
	principal = "User:manh.do"
	name = "system-platform-"
	pattern_type = "PREFIXED"
	provider = confluent-kafka.confluent
}
*/
func connectorsRBAC() *schema.Resource {
	return &schema.Resource{
		CreateContext: connectorsRBACCreate,
		DeleteContext: connectorsRBACDelete,
		ReadContext:   connectorsRBACRead,

		Schema: map[string]*schema.Schema{
			"principal": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Defined the principal - User or subject",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if !strings.Contains(v, "User:") || !strings.Contains(v, "Group:") && strings.Contains(v, "|") {
						errs = append(errs, fmt.Errorf("%q must be defined with User: or Group: and must not have |, got: %s", key, v))
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
				ValidateFunc: validation.StringInSlice(scopeRole, false),
			},
			"pattern_type": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringInSlice(validPatternType, false),
			},
			"name": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
			},
			"cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The ID of Kafka cluster",
			},
			"connect_cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The ID of Kafka Connect cluster",
			},
		},
	}
}

func connectorsRBACRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	principal := d.Get("principal").(string)
	role := d.Get("role").(string)
	clusterId := d.Get("cluster_id").(string)
	connectClusterId := d.Get("connect_cluster_id").(string)

	cDetails := &confluent.ClusterDetails{}
	cDetails.Clusters.KafkaCluster = clusterId
	cDetails.Clusters.ConnectCluster = connectClusterId

	roleBindings, err := c.LookupRoleBinding(principal, role, *cDetails)
	if err != nil {
		log.Printf("[ERROR] Error lookup role-binding %s from Confluent", err)
		return diag.FromErr(err)
	}

	r := make(map[string]string)

	for _, v := range roleBindings {
		if v.Name == d.Get("name").(string) {
			r[v.ResourceType] = v.PatternType
		}
	}

	if r["Connector"] != d.Get("pattern_type").(string) {
		err = fmt.Errorf("cannot find resource_type Subject of" + d.Get("name").(string))
		log.Printf("[ERROR] Error lookup role-binding from Confluent %s", err)
		return diag.FromErr(err)
	}
	return nil
}

func connectorsRBACCreate(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	principal := d.Get("principal").(string)
	role := d.Get("role").(string)
	clusterId:= d.Get("cluster_id").(string)
	connectClusterId := d.Get("connect_cluster_id").(string)


	cDetails := &confluent.ClusterDetails{
		Clusters: confluent.Clusters{
			KafkaCluster: clusterId,
			ConnectCluster: connectClusterId,
		},
	}

	u := confluent.RoleBinding{
		Scope: *cDetails,
		ResourcePatterns: []confluent.ResourcePattern{
			{
				ResourceType: "Connector",
				Name:         d.Get("name").(string),
				PatternType:  d.Get("pattern_type").(string),
			},
		},
	}

	err = c.IncreaseRoleBinding(principal, role, u)
	if err != nil {
		return diag.FromErr(err)
	}

	rId := clusterId + "|ConnectClusterId:" + connectClusterId + "|" + principal + "|" + role + "|Connector|" + d.Get("name").(string) + "|" + d.Get("pattern_type").(string)
	d.SetId(rId)
	return nil
}

func connectorsRBACDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	principal := d.Get("principal").(string)
	role := d.Get("role").(string)
	clusterId:= d.Get("cluster_id").(string)
	connectClusterId := d.Get("connect_cluster_id").(string)

	cDetails := &confluent.ClusterDetails{
		Clusters: confluent.Clusters{
			KafkaCluster: clusterId,
			ConnectCluster: connectClusterId,
		},
	}

	u := confluent.RoleBinding{
		Scope: *cDetails,
		ResourcePatterns: []confluent.ResourcePattern{
			{
				ResourceType: "Subject",
				Name:         d.Get("name").(string),
				PatternType:  d.Get("pattern_type").(string),
			},
		},
	}

	err = c.DecreaseRoleBinding(principal, role, u)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
