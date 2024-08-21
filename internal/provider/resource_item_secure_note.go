package provider

import (
	"github.com/geNAZt/terraform-provider-bitwarden/internal/bitwarden/bw"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceItemSecureNote() *schema.Resource {
	dataSourceItemSecureNoteSchema := baseSchema(Resource)

	return &schema.Resource{
		Description:   "Manages a secure note item.",
		CreateContext: createResource(bw.ObjectTypeItem, bw.ItemTypeSecureNote),
		ReadContext:   objectReadIgnoreMissing,
		UpdateContext: objectUpdate,
		DeleteContext: objectDelete,
		Importer:      importItemResource(bw.ObjectTypeItem, bw.ItemTypeSecureNote),
		Schema:        dataSourceItemSecureNoteSchema,
	}
}
