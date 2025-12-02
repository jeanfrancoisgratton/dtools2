// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/01 23:46
// Original filename: src/containers/info.go

package containers

import (
	"dtools2/rest"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

func InfoContainers(client *rest.Client, container string) error {
	// Preserve the current flag. For now, it might not be useful to do so, but might be, eventually
	currentRunningContainersFlag := OnlyRunningContainers
	currentExtendedContainersFlag := ExtendedContainerInfo
	currentDisplaySizeFlag := DisplaySizeValues
	OnlyRunningContainers = false
	ExtendedContainerInfo = true
	DisplaySizeValues = true

	// Fetch the container info from ListContainers()
	cs, err := ListContainers(client, false)
	if err != nil {
		OnlyRunningContainers = currentRunningContainersFlag
		ExtendedContainerInfo = currentExtendedContainersFlag
		DisplaySizeValues = currentDisplaySizeFlag
		return err
	}

	for _, ci := range cs {
		if ci.Names[0][1:] == container {
			// we got a match
			showExtendedInfo(ci)
			break
		}
	}

	// restore flags as they were before this fx call
	OnlyRunningContainers = currentRunningContainersFlag
	ExtendedContainerInfo = currentExtendedContainersFlag
	DisplaySizeValues = currentDisplaySizeFlag
	return nil
}

func showExtendedInfo(cInfo ContainerSummary) {
	state := ""
	switch cInfo.State {
	case "running":
		state = hftx.Green("running")
	case "paused":
	case "suspended":
	case "blocked":
		state = hftx.Yellow(cInfo.State)
	case "crashed":
		state = hftx.Red("crashed")
	default:
		state = hftx.White(strings.ToLower(cInfo.State))
	}
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Container name"), cInfo.Names[0][1:]))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Image"), getImageTag(cInfo.Image)))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Created"), time.Unix(cInfo.Created, 0).Format("2006.01.02 15:04:05")))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("State"), state))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Status"), strings.ToLower(cInfo.Status)))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("RW filesystem size"), formatImageSize(cInfo.SizeRw)))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("RootFS size"), formatImageSize(cInfo.SizeRootFs)))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Exposed ports"), prettifyPortsList(cInfo.Ports, ", ")))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Mount points"), prettifyMounts(cInfo.Mounts, ", ")))
	fmt.Fprintf(w, fmt.Sprintf("%s\t%s\n", hftx.Blue("Command"), cInfo.Command))
	w.Flush()
}
