// dtools2
// Written by J.F.Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/03/31 02:31
// Original filename: src/run_build/run_flag_types.go

package run_build

type UlimitFlag struct {
	Name string `json:"Name"`
	Soft int64  `json:"Soft"`
	Hard int64  `json:"Hard"`
}

type RestartPolicy struct {
	Name              string `json:"Name"`
	MaximumRetryCount int    `json:"MaximumRetryCount,omitempty"`
}
