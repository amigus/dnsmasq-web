package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type reservationData struct {
	Tags      []string `json:"tags"`
	IPv4      string   `json:"ipv4" binding:"required"`
	Hostname  string   `json:"hostname,omitempty"`
	LeaseTime string   `json:"lease_time,omitempty"`
}

type reservation struct {
	MAC string `json:"mac" binding:"required"`
	reservationData
}

func validateMAC(mac string) (*ipaddr.MACAddress, error) {
	addr, err := ipaddr.NewMACAddressString(mac).ToAddress()
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func validateIPv4(ipv4 string) (*ipaddr.IPAddress, error) {
	addr, err := ipaddr.NewIPAddressString(ipv4).ToAddress()
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func createReservationFile(input reservation, c *gin.Context, hostDir string, overwrite bool) {
	mac, err := validateMAC(input.MAC)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		return
	}

	content := mac.ToColonDelimitedString()
	
	if len(input.Tags) > 0 {
		content += "," + strings.Join(prefixTags(input.Tags), ",")
	}
	if ipv4, err := validateIPv4(input.IPv4); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid IPv4 address"})
		return
	} else {
		content += "," + ipv4.String()
	}
	if input.Hostname != "" {
		content += "," + input.Hostname
	}
	if input.LeaseTime != "" {
		content += "," + input.LeaseTime
	}
	content += "\n"

	filePath := filepath.Join(hostDir, mac.ToNormalizedString())
	if _, err := os.Stat(filePath); err == nil && !overwrite {
		c.JSON(http.StatusConflict, gin.H{"error": "exists"})
		return
	}
	if err := os.WriteFile(filePath, []byte(content), 0640); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "success"})
}

func updateReservationFile(input reservationData, mac string, c *gin.Context, hostDir string) {
	createReservationFile(reservation{MAC: mac, reservationData: input}, c, hostDir, true)
}

func deleteReservationFile(c *gin.Context, hostDir string) {
	mac, err := validateMAC(c.Param("mac"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		return
	}
	filePath := filepath.Join(hostDir, mac.ToNormalizedString())
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no such reservation"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func readReservationFile(mac string, hostDir string) (reservation, error) {
	filePath := filepath.Join(hostDir, mac)
	file, err := os.Open(filePath)
	if err != nil {
		return reservation{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			return reservation{}, fmt.Errorf("invalid file format")
		}

		res := reservation{
			MAC: parts[0],
			reservationData: reservationData{
				Tags:      []string{},
				IPv4:      parts[1],
				Hostname:  "",
				LeaseTime: "",
			},
		}
		if strings.HasPrefix(parts[1], "set:") {
			res.Tags = strings.Split(strings.Replace(parts[1], "set:", "", 1), ",")
			res.IPv4 = parts[2]
			if len(parts) > 3 {
				res.Hostname = parts[3]
			}
			if len(parts) > 4 {
				res.LeaseTime = parts[4]
			}
		} else {
			if len(parts) > 2 {
				res.Hostname = parts[2]
			}
			if len(parts) > 3 {
				res.LeaseTime = parts[3]
			}
		}

		return res, nil
	}

	if err := scanner.Err(); err != nil {
		return reservation{}, err
	}
	return reservation{}, nil
}

func getReservationFile(c *gin.Context, hostDir string) {
	macParam := c.Param("mac")
	if macParam != "" {
		mac, err := validateMAC(macParam)
		if err != nil || !mac.ToAddressString().IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		} else {
			if res, err := readReservationFile(mac.ToNormalizedString(), hostDir); err == nil {
				c.JSON(http.StatusOK, res)
			} else {
				if os.IsNotExist(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "no such reservation"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				}
			}
		}
	} else {
		var reservations []reservation
		err := filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				macAddr, err := validateMAC(info.Name())
				if err == nil {
					file, err := os.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()

					if res, err := readReservationFile(macAddr.ToNormalizedString(), hostDir); err == nil {
						reservations = append(reservations, res)
					}
				}
			}
			return nil
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, reservations)
	}
}

func prefixTags(tags []string) []string {
	for i, tag := range tags {
		tags[i] = "set:" + tag
	}
	return tags
}

func DhcpHostDir(r *gin.Engine, hostDir string) *gin.Engine {
	r.POST("/reservations", func(c *gin.Context) {
		var input struct {
			MAC string `json:"mac" binding:"required"`
			reservationData
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		createReservationFile(input, c, hostDir, false)
	})

	r.PUT("/reservations/:mac", func(c *gin.Context) {
		var input reservationData
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updateReservationFile(input, c.Param("mac"), c, hostDir)
	})

	r.DELETE("/reservations/:mac", func(c *gin.Context) {
		deleteReservationFile(c, hostDir)
	})

	r.GET("/reservations", func(c *gin.Context) {
		getReservationFile(c, hostDir)
	})

	r.GET("/reservations/:mac", func(c *gin.Context) {
		getReservationFile(c, hostDir)
	})

	return r
}
