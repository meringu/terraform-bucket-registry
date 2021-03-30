package example

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFoo() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFooRead,
		Schema: map[string]*schema.Schema{
			"bar": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceFooRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if err := d.Set("bar", "baz"); err != nil {
		return diag.FromErr(err)
	}

	// always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
