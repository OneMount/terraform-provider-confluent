package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	c "github.com/OneMount/terraform-provider-confluent/cplatform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: c.Provider})
}
