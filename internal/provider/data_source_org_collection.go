package provider

import (
	"github.com/geNAZt/terraform-provider-bitwarden/internal/bitwarden/bw"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOrgCollection() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get information on an existing organization collection.",
		ReadContext: readDataSourceObject(bw.ObjectTypeOrgCollection),
		Schema:      orgCollectionSchema(DataSource),
	}
}
