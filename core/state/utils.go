package coreState

import (
	"time"

	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// currentPoints calculates updated points based on current time.
func currentPoints(payer *Account, curTs int64) (*util.Uint128, error) {
	staking := payer.Staking
	used, err := payer.Staking.Sub(payer.Points)
	if err != nil {
		return nil, err
	}
	lastTs := payer.LastPointsTs
	elapsed := curTs - lastTs
	if time.Duration(elapsed)*time.Second >= PointsRegenerateDuration {
		return staking.DeepCopy(), nil
	}

	if elapsed < 0 {
		return nil, ErrElapsedTimestamp
	}

	if elapsed == 0 {
		return payer.Points.DeepCopy(), nil
	}

	// Points means remain points. So 0 means no used
	// regeneratedPoints = staking * elapsedTime / PointsRegenerateDuration
	// currentPoints = prevPoints + regeneratedPoints
	mul := util.NewUint128FromUint(uint64(elapsed))
	div := util.NewUint128FromUint(uint64(PointsRegenerateDuration / time.Second))
	v1, err := staking.Mul(mul)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to multiply uint128.")
		return nil, err
	}
	regen, err := v1.Div(div)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to divide uint128.")
		return nil, err
	}

	if regen.Cmp(used) >= 0 {
		return staking.DeepCopy(), nil
	}
	cur, err := payer.Points.Add(regen)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to add uint128.")
		return nil, err
	}
	return cur, nil
}
