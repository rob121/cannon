package content

import "github.com/rob121/cannon/internal/controllers"

func init() {
	controllers.Register(Definition(), New())
}
