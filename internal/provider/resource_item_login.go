package provider

import (
	"github.com/geNAZt/terraform-provider-bitwarden/internal/bitwarden/bw"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceItemLogin() *schema.Resource {
	dataSourceItemSecureNoteSchema := baseSchema(Resource)
	for k, v := range loginSchema(Resource) {
		dataSourceItemSecureNoteSchema[k] = v
	}

	return &schema.Resource{
		Description:   "Manages a login item.",
		CreateContext: createResource(bw.ObjectTypeItem, bw.ItemTypeLogin),
		ReadContext:   objectReadIgnoreMissing,
		UpdateContext: objectUpdate,
		DeleteContext: objectDelete,
		Importer:      importItemResource(bw.ObjectTypeItem, bw.ItemTypeLogin),
		Schema:        dataSourceItemSecureNoteSchema,
	}
}
