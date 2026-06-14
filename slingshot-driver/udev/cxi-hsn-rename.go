package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type hsnDevice struct {
	pciAddr string
	netif   string
}

var hsnDeviceIDs = [][2]string{
	{"0x15b3", "0x1017"}, // Mellanox ConnectX-5
	{"0x17db", "0x0501"}, // HPE Cray Slingshot SS-11 1P
}

func readSysfs(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func isHSNDevice(vendor, device string) bool {
	for _, id := range hsnDeviceIDs {
		if vendor == id[0] && device == id[1] {
			return true
		}
	}
	return false
}

func findHSNDevices() []hsnDevice {
	var devices []hsnDevice

	pciPaths, _ := filepath.Glob("/sys/devices/pci*/*:*:*.*")
	subPaths, _ := filepath.Glob("/sys/devices/pci*/*:*:*.*/????:??:??.?")
	pciPaths = append(pciPaths, subPaths...)

	sort.Strings(pciPaths)

	for _, pciDev := range pciPaths {
		class := readSysfs(filepath.Join(pciDev, "class"))
		if class != "0x020000" {
			continue
		}
		vendor := readSysfs(filepath.Join(pciDev, "vendor"))
		device := readSysfs(filepath.Join(pciDev, "device"))
		if !isHSNDevice(vendor, device) {
			continue
		}

		netDir := filepath.Join(pciDev, "net")
		entries, err := os.ReadDir(netDir)
		if err != nil || len(entries) == 0 {
			// Try one level deeper (e.g. virtio)
			subDirs, _ := os.ReadDir(pciDev)
			for _, sub := range subDirs {
				netDir = filepath.Join(pciDev, sub.Name(), "net")
				entries, err = os.ReadDir(netDir)
				if err == nil && len(entries) > 0 {
					break
				}
			}
		}
		if len(entries) == 0 {
			continue
		}

		devices = append(devices, hsnDevice{
			pciAddr: pciDev,
			netif:   entries[0].Name(),
		})
	}

	return devices
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <netif>\n", os.Args[0])
		os.Exit(1)
	}

	match := os.Args[1]
	devices := findHSNDevices()

	for i, dev := range devices {
		if dev.netif == match {
			fmt.Printf("hsn%d\n", i)
			os.Exit(0)
		}
	}

	os.Exit(1)
}
