package health

import (
	"errors"
	"fmt"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/helper"
	"github.com/hectorgimenez/koolo/internal/hid"
	"github.com/hectorgimenez/koolo/internal/stats"
	"go.uber.org/zap"
	"time"
)

var ErrDied = errors.New("you died :(")
var ErrChicken = errors.New("chicken")
var ErrMercChicken = errors.New("mercenary chicken")

const (
	healingInterval     = time.Second * 8
	healingMercInterval = time.Second * 8
	manaInterval        = time.Second * 8
	rejuvInterval       = time.Second * 2
)

// Manager responsibility is to keep our character and mercenary alive, monitoring life and giving potions when needed
type Manager struct {
	logger        *zap.Logger
	beltManager   BeltManager
	lastRejuv     time.Time
	lastRejuvMerc time.Time
	lastHeal      time.Time
	lastMana      time.Time
	lastMercHeal  time.Time
}

func NewHealthManager(logger *zap.Logger, beltManager BeltManager) Manager {
	return Manager{
		logger:      logger,
		beltManager: beltManager,
	}
}

func (hm *Manager) HandleHealthAndMana(d game.Data) error {
	hpConfig := config.Config.Health
	// Safe area, skipping
	if d.Area.IsTown() {
		return nil
	}

	status := d.Health

	if status.Life == 0 {
		// After dying we need to press esc and wait the loading screen until we can exit game, it's a bit hacky but it works
		helper.Sleep(1000)
		hid.PressKey("esc")
		helper.Sleep(10000)
		stats.FinishCurrentRun(stats.EventDeath)
		return ErrDied
	}

	usedRejuv := false
	if time.Since(hm.lastRejuv) > rejuvInterval && (status.HPPercent() <= hpConfig.RejuvPotionAtLife || status.MPPercent() < hpConfig.RejuvPotionAtMana) {
		usedRejuv = hm.beltManager.DrinkPotion(d, game.RejuvenationPotion, false)
		if usedRejuv {
			hm.lastRejuv = time.Now()
		}
	}

	if !usedRejuv {
		if status.HPPercent() <= hpConfig.ChickenAt {
			stats.FinishCurrentRun(stats.EventChicken)
			return fmt.Errorf("%w: Current Health: %d (%d percent)", ErrChicken, status.Life, status.HPPercent())
		}

		if status.HPPercent() <= hpConfig.HealingPotionAt && time.Since(hm.lastHeal) > healingInterval {
			hm.beltManager.DrinkPotion(d, game.HealingPotion, false)
			hm.lastHeal = time.Now()
		}

		if status.MPPercent() <= hpConfig.ManaPotionAt && time.Since(hm.lastMana) > manaInterval {
			hm.beltManager.DrinkPotion(d, game.ManaPotion, false)
			hm.lastMana = time.Now()
		}
	}

	// Mercenary
	if status.Merc.Alive {
		usedMercRejuv := false
		if time.Since(hm.lastRejuvMerc) > rejuvInterval && status.MercHPPercent() <= hpConfig.MercRejuvPotionAt {
			usedMercRejuv = hm.beltManager.DrinkPotion(d, game.RejuvenationPotion, true)
			if usedMercRejuv {
				hm.lastRejuvMerc = time.Now()
			}
		}

		if !usedMercRejuv {
			if status.MercHPPercent() <= hpConfig.MercChickenAt {
				stats.FinishCurrentRun(stats.EventMercChicken)
				return fmt.Errorf("%w: Current Merc Health: %d (%d percent)", ErrMercChicken, status.Merc.Life, status.MercHPPercent())
			}

			if status.MercHPPercent() <= hpConfig.MercHealingPotionAt && time.Since(hm.lastMercHeal) > healingMercInterval {
				hm.beltManager.DrinkPotion(d, game.HealingPotion, true)
				hm.lastMercHeal = time.Now()
			}
		}
	}

	return nil
}
