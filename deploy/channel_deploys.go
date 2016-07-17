package deploy

import "time"

type ChannelDeploys struct {
	store Store
}

func NewChannelDeploys(store Store) *ChannelDeploys {
	return &ChannelDeploys{store: store}
}

func (repo *ChannelDeploys) Current(channelID string) (Deploy, bool) {
	return repo.store.Get(channelID)
}

func (repo *ChannelDeploys) Start(channelID string, d Deploy) (Deploy, bool) {
	for {
		current, ok := repo.Current(channelID)
		if !ok {
			break
		}

		if current.User.ID != d.User.ID {
			return current, false
		}

		repo.Finish(channelID)
	}

	d.StartedAt = time.Now()
	repo.store.Set(channelID, d)

	return d, true
}

func (repo *ChannelDeploys) Finish(channelID string) (Deploy, bool) {
	return repo.store.Del(channelID)
}
