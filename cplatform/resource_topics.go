package cplatform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	confluent "github.com/OneMount/gonfluent"
)

func topics() *schema.Resource {
	return &schema.Resource{
		CreateContext: topicsCreate,
		DeleteContext: topicsDelete,
		ReadContext:   topicsRead,
		UpdateContext: topicsUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"partitions": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    false,
				Description: "Number of partitions.",
			},
			"replication_factor": {
				Type:        schema.TypeInt,
				ForceNew:    false,
				Optional:    true,
				Description: "Number of replication factors.",
			},
			"config": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "A map of string k/v attributes.",
				Elem:        schema.TypeString,
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

func topicsRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	clusterId := d.Get("cluster_id").(string)
	topicName := d.Id()
	//topicName := d.Get("topic_name").(string)

	topic, err := c.GetTopic(clusterId, topicName)
	if err != nil {
		log.Printf("[ERROR] Error getting topics %s from Confluent", err)
		d.SetId("")
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Setting the state from Confluent %v", topic)
	err = d.Set("name", topic.Name)

	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}

func topicsCreate(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Creating topic" + d.Get("name").(string))
	c := meta.(*confluent.Client)
	topicName := d.Get("name").(string)

	if err := topicCreateFunc(c, d); err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Creat topic" + d.Get("name").(string) + "successfully!")
	d.SetId(topicName)

	return diags
}

func topicCreateFunc(c *confluent.Client, d *schema.ResourceData) error {
	clusterId:= d.Get("cluster_id").(string)
	topicName := d.Get("name").(string)
	partitionsCount := d.Get("partitions").(int)
	replicationFactor := d.Get("replication_factor").(int)
	config := d.Get("config").(map[string]interface{})

	if confluentPlacementConstraintsIsPresent(d) {
		replicationFactor = 0
	}

	var topicConfigs []confluent.TopicConfig

	for key, value := range config {
		switch value := value.(type) {
		case string:
			topicConfigs = append(topicConfigs, confluent.TopicConfig{
				Name:  key,
				Value: value,
			})
		}
	}

	err = c.CreateTopic(clusterId, topicName, partitionsCount, replicationFactor, topicConfigs, nil)
	if err != nil {
		return err
	}
	d.SetId(topicName)
	return nil
}

func topicsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Delete topic" + d.Get("name").(string))
	c := meta.(*confluent.Client)

	clusterId:= d.Get("cluster_id").(string)
	topicName := d.Id()

	if err := c.DeleteTopic(clusterId, topicName); err != nil {
		return diag.FromErr(err)
	}

	if err := waitForTopicDelete(ctx, c, d.Id(), clusterId); err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Topic" + d.Get("name").(string) + " deleted!")
	return diags
}

func topicsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	clusterId:= d.Get("cluster_id").(string)
	if err != nil {
		return diag.FromErr(err)
	}
	t := confluent.Topic{
		Name: d.Id(),
	}

	// update replica count of existing partitions before adding new ones
	if d.HasChange("replication_factor") {
		if !(confluentPlacementConstraintsIsPresent(d)) {
			oi, ni := d.GetChange("replication_factor")
			oldRF := oi.(int)
			newRF := ni.(int)
			t.ReplicationFactor = int16(newRF)
			log.Printf("[INFO] Updating replication_factor from %d to %d", oldRF, newRF)
			err := c.UpdateReplicationsFactor(t)
			if err != nil {
				return diag.FromErr(err)
			}

			if err := waitForRFUpdate(ctx, c, d.Id()); err != nil {
				return diag.FromErr(err)
			}
			if err := waitForTopicRefresh(ctx, c, d.Id(), clusterId, t); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("partitions") {
		// update should only be called when we're increasing partitions
		oi, ni := d.GetChange("partitions")
		oldPartitions := oi.(int)
		newPartitions := ni.(int)
		if newPartitions < oldPartitions {
			return diag.FromErr(fmt.Errorf("cannot decrease the number of partitions of topic"))
		}
		log.Printf("[INFO] Updating partitions from %d to %d", oldPartitions, newPartitions)
		t.Partitions = int32(newPartitions)

		if err := c.UpdatePartitions(t); err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTopicRefresh(ctx, c, d.Id(), clusterId, t); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("config") {
		config := d.Get("config").(map[string]interface{})

		var topicConfigs []confluent.TopicConfig

		for key, value := range config {
			switch value := value.(type) {
			case string:
				topicConfigs = append(topicConfigs, confluent.TopicConfig{
					Name:  key,
					Value: value,
				})
			}
		}

		if err := c.UpdateTopicConfigs(clusterId, d.Id(),topicConfigs); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := waitForTopicRefresh(ctx, c, d.Id(), clusterId, t); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func confluentPlacementConstraintsIsPresent(d *schema.ResourceData) bool {
	config := d.Get("config").(map[string]interface{})
	for config["confluent.placement.constraints"] != nil {
		return true
	}
	return false
}

func waitForRFUpdate(ctx context.Context, c *confluent.Client, topic string) error {
	refresh := func() (interface{}, string, error) {
		isRFUpdating, err := c.IsReplicationFactorUpdating(topic)
		if err != nil {
			return nil, "Error", err
		} else if isRFUpdating {
			return nil, "Updating", nil
		} else {
			return "not-nil", "Ready", nil
		}
	}

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"Updating"},
		Target:       []string{"Ready"},
		Refresh:      refresh,
		Timeout:      120 * time.Second,
		Delay:        1 * time.Second,
		PollInterval: 1 * time.Second,
		MinTimeout:   2 * time.Second,
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf(
			"error waiting for topic (%s) replication_factor to update: %s",
			topic, err)
	}

	return nil
}

func waitForTopicDelete(ctx context.Context,c *confluent.Client, topic, clusterId string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{"Updating"},
		Target:       []string{"Ready"},
		Refresh:      topicDeleteRefreshFunc(c, topic, clusterId),
		Timeout:      120 * time.Second,
		Delay:        1 * time.Second,
		PollInterval: 1 * time.Second,
		MinTimeout:   2 * time.Second,
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf(
			"error waiting for topic (%s) to become ready: %s",
			topic, err)
	}

	return nil
}

func waitForTopicRefresh(ctx context.Context,c *confluent.Client, topic, clusterId string, expected confluent.Topic) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{"Updating"},
		Target:       []string{"Ready"},
		Refresh:      topicRefreshFunc(c, topic, clusterId, expected),
		Timeout:      120 * time.Second,
		Delay:        1 * time.Second,
		PollInterval: 1 * time.Second,
		MinTimeout:   2 * time.Second,
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf(
			"error waiting for topic (%s) to become ready: %s",
			topic, err)
	}

	return nil
}

func topicRefreshFunc(c *confluent.Client, topic, clusterId string, expected confluent.Topic) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		log.Printf("[DEBUG] waiting for topic to update %s", topic)
		actual, err := c.GetTopic(clusterId, topic)
		if err != nil {
			log.Printf("[ERROR] could not read topic %s, %s", topic, err)
			return actual, "Error", err
		}
		p, err  := c.GetTopicPartitions(clusterId, topic)
		if err != nil {
			log.Printf("[ERROR] could not read topic %s, %s", topic, err)
			return actual, "Error", err
		}

		if int16(actual.ReplicationFactor) == expected.ReplicationFactor && int32(len(p)) == expected.Partitions  {
			return actual, "Ready", nil
		}

		return nil, fmt.Sprintf("%v != %v", "", ""), nil
	}
}

func topicDeleteRefreshFunc(c *confluent.Client, topic, clusterId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		log.Printf("[DEBUG] waiting for topic to delete %s", topic)
		actual, err := c.GetTopic(clusterId, topic)
		if err != nil && strings.Contains(err.Error(), "404") {
			return actual, "Ready", nil
		}

		return nil, fmt.Sprintf("cannot delete topic: %v", topic), nil
	}
}

