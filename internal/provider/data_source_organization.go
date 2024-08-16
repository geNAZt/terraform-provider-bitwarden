package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/paymenttools/terraform-provider-bitwarden/internal/bitwarden/bw"
)

func dataSourceOrganization() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get information on an existing organization.",
		ReadContext: readDataSourceObject(bw.ObjectTypeOrganization),
		Schema:      organizationSchema(),
	}
}
