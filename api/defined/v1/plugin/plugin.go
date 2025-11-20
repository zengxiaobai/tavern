package plugin

import "github.com/omalloc/tavern/contrib/transport"

type Plugin interface {
	transport.Server
}
