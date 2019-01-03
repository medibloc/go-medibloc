package core

import "github.com/medibloc/go-medibloc/util"

//Bandwidth is structure for cpu and net bandwidth
type Bandwidth struct {
	cpuUsage uint64
	netUsage uint64
}

//NewBandwidth returns new bandwidth
func NewBandwidth(cpu, net uint64) *Bandwidth {
	return &Bandwidth{
		cpuUsage: cpu,
		netUsage: net,
	}
}

func (b *Bandwidth) Clone() *Bandwidth {
	return &Bandwidth{
		cpuUsage: b.cpuUsage,
		netUsage: b.netUsage,
	}
}

//Add add obj's bandwidth
func (b *Bandwidth) Add(obj *Bandwidth) {
	b.cpuUsage += obj.cpuUsage
	b.netUsage += obj.netUsage
}

//Sub subtract obj's bandwidth
func (b *Bandwidth) Sub(obj *Bandwidth) {
	b.cpuUsage -= obj.cpuUsage
	b.netUsage -= obj.netUsage
}

//CalcPoints multiply bandwidth and price
func (b *Bandwidth) CalcPoints(p Price) (points *util.Uint128, err error) {
	cpuPoints, err := p.cpuPrice.Mul(util.NewUint128FromUint(b.cpuUsage))
	if err != nil {
		return nil, err
	}
	netPoints, err := p.netPrice.Mul(util.NewUint128FromUint(b.netUsage))
	if err != nil {
		return nil, err
	}
	return cpuPoints.Add(netPoints)
}

//IsZero return true when cpu and net value are both zero
func (b Bandwidth) IsZero() bool {
	return b.cpuUsage == 0 && b.netUsage == 0
}

//Price is structure for prices of cpu and net
type Price struct {
	cpuPrice *util.Uint128
	netPrice *util.Uint128
}
