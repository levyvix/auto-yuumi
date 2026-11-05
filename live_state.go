package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type ownLife struct{ Known, Dead bool }

func parseOwnLife(reader io.Reader) (ownLife, error) {
	var data struct {
		ActivePlayer struct {
			ChampionStats struct {
				CurrentHealth *float64 `json:"currentHealth"`
				MaxHealth     float64  `json:"maxHealth"`
			} `json:"championStats"`
		} `json:"activePlayer"`
		AllPlayers []json.RawMessage `json:"allPlayers"`
	}
	if err := json.NewDecoder(reader).Decode(&data); err != nil {
		return ownLife{}, err
	}
	stats := data.ActivePlayer.ChampionStats
	if len(data.AllPlayers) == 0 || stats.CurrentHealth == nil || stats.MaxHealth <= 0 {
		return ownLife{}, nil
	}
	return ownLife{Known: true, Dead: *stats.CurrentHealth <= 0}, nil
}

// The game serves a self-signed certificate on this fixed loopback endpoint.
var localGameClient = &http.Client{Timeout: 500 * time.Millisecond, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

type lifeCache struct {
	checked time.Time
	state   ownLife
}

func (c *lifeCache) read(now time.Time) ownLife {
	if now.Sub(c.checked) < time.Second {
		return c.state
	}
	c.checked = now
	c.state = ownLife{}
	response, err := localGameClient.Get("https://127.0.0.1:2999/liveclientdata/allgamedata")
	if err != nil {
		return c.state
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return c.state
	}
	state, err := parseOwnLife(io.LimitReader(response.Body, 2<<20))
	if err == nil {
		c.state = state
	}
	return c.state
}

type purchaseEvents struct {
	deadHandled, baseHandled bool
	baseFrames, awayFrames   int
}

func (p *purchaseEvents) next(life ownLife, homeguard bool) string {
	if !life.Known {
		return ""
	}
	if !life.Dead {
		p.deadHandled = false
	}
	if homeguard {
		p.baseFrames++
		p.awayFrames = 0
	} else {
		p.baseFrames = 0
		p.awayFrames++
		if p.awayFrames >= 3 {
			p.baseHandled = false
		}
	}
	if life.Dead {
		p.baseHandled = false
		p.baseFrames = 0
	}
	if life.Dead && !p.deadHandled {
		p.deadHandled = true
		return "morte"
	}
	if !life.Dead && p.baseFrames >= 3 && !p.baseHandled {
		p.baseHandled = true
		return "chegada a base"
	}
	return ""
}
