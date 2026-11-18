package routeros

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// isMaskedSecret reports whether v is a non-empty string of asterisks, the way RouterOS
// returns write-only secrets.
func isMaskedSecret(v string) bool {
	return v != "" && strings.Trim(v, "*") == ""
}

// MaskedSecretNotInConfig is a DiffSuppressFunc for write-only secrets that RouterOS
// returns masked ("*****") or empty. It suppresses the diff only when the attribute is
// absent from the configuration (new == "") and the stored value is empty or consists
// of asterisks only. A secret the user did set (new != "") always produces a diff.
//
// This only covers the masked case: RouterOS returns the mask when the provider's
// RouterOS user lacks the `sensitive` policy. Then an absent or empty secret in the
// configuration means "not managed": the router keeps its value, the secret cannot be
// cleared through the configuration, and the serializer (TerraformResourceDataToMikrotik)
// does not send the mask back (it sends a masked value only when the configuration
// holds exactly that value). With the `sensitive` policy RouterOS returns the real
// secret: an absent or empty secret is then a normal diff to "", and apply clears it.
func MaskedSecretNotInConfig(_, old, new string, _ *schema.ResourceData) bool {
	return new == "" && strings.Trim(old, "*") == ""
}
