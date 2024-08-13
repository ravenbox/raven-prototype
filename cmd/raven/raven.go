package raven

import (
	"github.com/ravenbox/raven-prototype/pkg/memstore"
	"github.com/ravenbox/raven-prototype/pkg/types"
)

func main() {
	storage, err := memstore.NewStorage()
	if err != nil {
		panic(err)
	}

	storage.NewNest(&types.Nest{
		Id:   "a",
		Name: "Cozy Nest",
		StreamChannel: types.StreamChannel{
			Id:   "a",
			Name: "First Stream Channel",
		},
		TextChannel: types.TextChannel{
			Id:   "a",
			Name: "First Text Channel",
		}},
	)
}
