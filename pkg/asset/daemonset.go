package asset

func (a *Asset) DaemonSet() *daemonset {
	return &daemonset{
		asset: a,
	}
}

type daemonset struct {
	asset *Asset
}

func (d *daemonset) Name() string {
	return d.asset.Values().Name
}
