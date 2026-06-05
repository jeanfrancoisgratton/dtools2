// dtools2
// Written by J.F.Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/03/31 02:22
// Original filename: src/run_build/run_flag_helpers.go

package run_build

import (
	"fmt"
	"strconv"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// This file contains the various flag parsers needed by dtools run

func parseUlimits(input []string) ([]UlimitFlag, error) {
	var out []UlimitFlag

	for _, u := range input {
		// format: name=soft[:hard]
		parts := strings.SplitN(u, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid ulimit format: %s", u)
		}

		name := parts[0]
		vals := strings.Split(parts[1], ":")

		var soft, hard int64
		var err error

		if len(vals) == 1 {
			soft, err = strconv.ParseInt(vals[0], 10, 64)
			if err != nil {
				return nil, err
			}
			hard = soft
		} else if len(vals) == 2 {
			soft, err = strconv.ParseInt(vals[0], 10, 64)
			if err != nil {
				return nil, err
			}
			hard, err = strconv.ParseInt(vals[1], 10, 64)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid ulimit format: %s", u)
		}

		out = append(out, UlimitFlag{
			Name: name,
			Soft: soft,
			Hard: hard,
		})
	}

	return out, nil
}

func parseBytes(input string) (int64, error) {
	if input == "" {
		return 0, nil
	}

	mult := int64(1)
	last := input[len(input)-1]

	switch last {
	case 'k', 'K':
		mult = 1024
		input = input[:len(input)-1]
	case 'm', 'M':
		mult = 1024 * 1024
		input = input[:len(input)-1]
	case 'g', 'G':
		mult = 1024 * 1024 * 1024
		input = input[:len(input)-1]
	}

	val, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return 0, err
	}

	return val * mult, nil
}

func parseRestartPolicy(input string) (RestartPolicy, error) {
	if input == "" {
		return RestartPolicy{}, nil
	}

	parts := strings.Split(input, ":")
	name := parts[0]

	switch name {
	case "no", "always", "unless-stopped":
		return RestartPolicy{Name: name}, nil

	case "on-failure":
		policy := RestartPolicy{Name: name}

		if len(parts) == 2 {
			n, err := strconv.Atoi(parts[1])
			if err != nil {
				return policy, err
			}
			policy.MaximumRetryCount = n
		}

		return policy, nil

	default:
		return RestartPolicy{}, fmt.Errorf("invalid restart policy: %s", input)
	}
}

func getRunFlagValues() (*HostConfig, *ce.CustomError) {
	hc := &HostConfig{}

	ulimits, err := parseUlimits(RunUlimits)
	if err != nil {
		return nil, &ce.CustomError{Title: "Invalid --ulimit flag", Message: err.Error()}
	}
	hc.Ulimits = ulimits

	// Memory
	mem, err := parseBytes(RunMemory)
	if err != nil {
		return nil, &ce.CustomError{Title: "Invalid --memory flag", Message: err.Error()}
	}
	hc.Memory = mem

	// CPUs → NanoCPUs (Docker expects 1e9 units per CPU)
	if RunCPUs > 0 {
		hc.NanoCPUs = int64(RunCPUs * 1e9)
	}

	// CPU shares
	if RunCPUShares > 0 {
		hc.CpuShares = RunCPUShares
	}

	// Restart policy
	rp, err := parseRestartPolicy(RunRestart)
	if err != nil {
		return nil, &ce.CustomError{Title: "Invalid --restart flag", Message: err.Error()}
	}
	hc.RestartPolicy = rp

	// Privileged
	hc.Privileged = RunPrivileged

	// Capabilities
	hc.CapAdd = RunCapAdd
	hc.CapDrop = RunCapDrop

	// Read-only rootfs
	hc.ReadonlyRootfs = RunReadOnly

	// shm-size
	shm, err := parseBytes(RunShmSize)
	if err != nil {
		return nil, &ce.CustomError{Title: "Invalid --shm-size flag", Message: err.Error()}
	}
	if shm > 0 {
		hc.ShmSize = shm
	}

	// PIDs limit
	if RunPidsLimit > 0 {
		hc.PidsLimit = RunPidsLimit
	}
	return hc, nil
}
