package memstore

import "github.com/ravenbox/raven-prototype/pkg/types"

type Storage struct {
	Users          []types.User          `json:"users"`
	Trees          []types.Tree          `json:"trees"`
	Nests          []types.Nest          `json:"nests"`
	StreamChannels []types.StreamChannel `json:"stream_channels"`
	TextChannels   []types.TextChannel   `json:"text_channels"`
}

func NewStorage() (*Storage, error) {
	return &Storage{}, nil
}

func (s *Storage) NewUser(user *types.User) error {
	s.Users = append(s.Users, *user)
	return nil // TODO: there should be SOME error to return here
}

func (s *Storage) NewTree(tree *types.Tree) error {
	s.Trees = append(s.Trees, *tree)
	return nil // TODO: there should be SOME error to return here
}

func (s *Storage) NewNest(nest *types.Nest) error {
	s.Nests = append(s.Nests, *nest)
	return nil // TODO: there should be SOME error to return here
}

func (s *Storage) NewStreamChannel(streamChannel *types.StreamChannel) error {
	s.StreamChannels = append(s.StreamChannels, *streamChannel)
	return nil // TODO: there should be SOME error to return here
}

func (s *Storage) NewTextChannel(textChannel *types.TextChannel) error {
	s.TextChannels = append(s.TextChannels, *textChannel)
	return nil // TODO: there should be SOME error to return here
}
