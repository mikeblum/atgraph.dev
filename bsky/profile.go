package bsky

import (
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/bsky"
)

type Profile struct {
	*bsky.ActorGetProfiles_Output
}

func (p Profile) String() string {
	var sb strings.Builder
	for _, profile := range p.Profiles {
		sb.WriteString(fmt.Sprintf("handle=%s did=%s", profile.Handle, profile.Did))
	}
	return sb.String()
}
