package server

import (
	"encoding/json"
	"github.com/Adeithe/go-twitch/pubsub"
	"log"
	"ttv-cli/internal/pkg/twitch/gql/query/channel"
	"ttv-cli/internal/pkg/twitch/pubsub/communitypointschannel"
)

type reward struct {
	Title             string `json:"title"`
	CooldownExpiresAt string `json:"cooldown_expires_at"`
	Id                string `json:"id"`
	Cost              int    `json:"cost"`
}

func (r *reward) toBytes() []byte {
	bytes, err := json.Marshal(r)
	if err != nil {
		log.Fatalf("Unable to convert reward to bytes: %v, error: %s\n", r, err)
	}
	return bytes
}

// pumpEvents pumps redemption events from Twitch to Clients
func (h *Hub) pumpEvents(streamer string) {

	c, err := channel.GetChannel(streamer)
	if err != nil {
		log.Fatalln("pumpEvents: ", err)
	}

	go func() {
		for _, customReward := range c.CommunityPointsSettings.CustomRewards {
			if !customReward.IsEnabled || customReward.IsPaused {
				continue
			}
			h.rewardsById.Store(customReward.Id, &reward{
				Title:             customReward.Title,
				CooldownExpiresAt: customReward.CooldownExpiresAt,
				Id:                customReward.Id,
				Cost:              customReward.Cost,
			})
		}
	}()

	go func() {
		p := pubsub.New()
		err := p.Listen("community-points-channel-v1", c.Id)
		if err != nil {
			log.Fatalln("Could not subscribe to community-points-channel-v1: ", err)
		}

		p.OnShardMessage(func(shard int, topic string, data []byte) {
			var response communitypointschannel.Response
			if err := json.Unmarshal(data, &response); err != nil {
				log.Fatalln(err)
			}
			if response.Type == "custom-reward-updated" {
				updatedReward := response.Data.UpdatedReward
				r := &reward{
					Title:             updatedReward.Title,
					Id:                updatedReward.Id,
					CooldownExpiresAt: updatedReward.CooldownExpiresAt,
					Cost:              updatedReward.Cost,
				}
				h.rewardsById.Store(updatedReward.Id, r)
				h.broadcast <- r.toBytes()
			}
		})
	}()
}