package models

import (
	"github.com/snakem982/pandora-box/pkg/utils"
	"math/big"
	"time"
)

type Profile struct {
	Id        string   `json:"id"`
	Type      int      `json:"type"` // 1: 远程订阅 2：本地配置 3：爬取合并
	Title     string   `json:"title,omitempty"`
	Order     string   `json:"order"`
	Selected  bool     `json:"selected,omitempty"`
	Path      string   `json:"path"`
	Content   string   `json:"content,omitempty"`
	Used      *big.Int `json:"used,omitempty"`
	Available *big.Int `json:"available,omitempty"`
	Total     *big.Int `json:"total,omitempty"`
	Expire    string   `json:"expire,omitempty"`
	Interval  string   `json:"interval,omitempty"`
	Home      string   `json:"home,omitempty"`
	Support   string   `json:"support,omitempty"`
	Update    string   `json:"update,omitempty"`
	Template  string   `json:"template,omitempty"`
}

func (p *Profile) GetUpdateTime() time.Time {
	dateTime, _ := utils.ParseDateTime(p.Update)
	return dateTime
}

func (p *Profile) SetUpdateTime() {
	p.Update = utils.GetDateTime()
}
