package reservations

import (
	"fmt"
	"strings"
)

type FileRecord struct {
	MAC       string
	Tags      []string
	IPv4      string
	Hostname  string
	LeaseTime string
}

func Format(record FileRecord) string {
	fields := []string{record.MAC}
	for _, tag := range record.Tags {
		fields = append(fields, "set:"+tag)
	}
	fields = append(fields, record.IPv4)
	if record.Hostname != "" || record.LeaseTime != "" {
		fields = append(fields, record.Hostname)
	}
	if record.LeaseTime != "" {
		fields = append(fields, record.LeaseTime)
	}
	return strings.Join(fields, ",")
}

func Parse(line string) (FileRecord, error) {
	parts := strings.Split(strings.TrimSpace(line), ",")
	if len(parts) < 2 {
		return FileRecord{}, fmt.Errorf("invalid file format")
	}
	index := 1
	tags := make([]string, 0, len(parts)-1)
	for index < len(parts) && strings.HasPrefix(parts[index], "set:") {
		tags = append(tags, strings.TrimPrefix(parts[index], "set:"))
		index++
	}
	if index >= len(parts) {
		return FileRecord{}, fmt.Errorf("invalid file format")
	}
	record := FileRecord{
		MAC:  parts[0],
		Tags: tags,
		IPv4: parts[index],
	}
	index++
	if index < len(parts) {
		record.Hostname = parts[index]
		index++
	}
	if index < len(parts) {
		record.LeaseTime = parts[index]
		index++
	}
	if index != len(parts) {
		return FileRecord{}, fmt.Errorf("invalid file format")
	}
	return record, nil
}