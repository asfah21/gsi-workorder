package usersync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func parseUsers(rows []map[string]json.RawMessage) ([]OdooUser, error) {
	users := make([]OdooUser, 0, len(rows))
	for _, row := range rows {
		id, err := decodeInt64Field(row, "id")
		if err != nil {
			return nil, err
		}
		login, err := decodeStringField(row, "login")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(login) == "" {
			return nil, fmt.Errorf("%w: login for user %d", errMissingField, id)
		}
		active, err := decodeBoolField(row, "active")
		if err != nil {
			return nil, err
		}
		name, err := decodeOptionalString(row, "name")
		if err != nil {
			return nil, err
		}
		email, err := decodeOptionalString(row, "email")
		if err != nil {
			return nil, err
		}

		users = append(users, OdooUser{
			OdooID: id,
			Login:  strings.TrimSpace(login),
			Name:   strings.TrimSpace(name),
			Email:  strings.TrimSpace(email),
			Active: active,
		})
	}
	return users, nil
}

func parseEmployees(rows []map[string]json.RawMessage, nikField string) ([]OdooEmployee, error) {
	employees := make([]OdooEmployee, 0, len(rows))
	for _, row := range rows {
		id, err := decodeInt64Field(row, "id")
		if err != nil {
			return nil, err
		}
		active, err := decodeBoolField(row, "active")
		if err != nil {
			return nil, err
		}
		nik, err := decodeStringField(row, nikField)
		if err != nil {
			return nil, fmt.Errorf("employee %d field %q: %w", id, nikField, err)
		}
		name, err := decodeOptionalString(row, "name")
		if err != nil {
			return nil, err
		}
		jobTitle, err := decodeOptionalString(row, "job_title")
		if err != nil {
			return nil, err
		}
		_, department, err := decodeOptionalMany2One(row, "department_id")
		if err != nil {
			return nil, err
		}
		_, siteLocation, err := decodeOptionalMany2One(row, "work_location_id")
		if err != nil {
			return nil, err
		}
		userID, _, err := decodeOptionalMany2One(row, "user_id")
		if err != nil {
			return nil, err
		}

		employees = append(employees, OdooEmployee{
			OdooID:       id,
			NIK:          strings.TrimSpace(nik),
			Name:         strings.TrimSpace(name),
			JobTitle:     strings.TrimSpace(jobTitle),
			Department:   strings.TrimSpace(department),
			SiteLocation: strings.TrimSpace(siteLocation),
			OdooUserID:   userID,
			Active:       active,
		})
	}
	return employees, nil
}

func decodeStringField(row map[string]json.RawMessage, key string) (string, error) {
	raw, ok := row[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", errMissingField, key)
	}
	return decodeString(raw)
}

func decodeOptionalString(row map[string]json.RawMessage, key string) (string, error) {
	raw, ok := row[key]
	if !ok {
		return "", nil
	}
	return decodeString(raw)
}

func decodeBoolField(row map[string]json.RawMessage, key string) (bool, error) {
	raw, ok := row[key]
	if !ok {
		return false, fmt.Errorf("%w: %s", errMissingField, key)
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("field %s is not a boolean", key)
	}
	return value, nil
}

func decodeInt64Field(row map[string]json.RawMessage, key string) (int64, error) {
	raw, ok := row[key]
	if !ok {
		return 0, fmt.Errorf("%w: %s", errMissingField, key)
	}
	return decodeInt64(raw)
}

func decodeString(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "false" {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("value is not a string")
	}
	return value, nil
}

func decodeInt64(raw json.RawMessage) (int64, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var number json.Number
	if err := decoder.Decode(&number); err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}
	value, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}
	return value, nil
}

func decodeOptionalMany2One(row map[string]json.RawMessage, key string) (*int64, string, error) {
	raw, ok := row[key]
	if !ok {
		return nil, "", nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "false" {
		return nil, "", nil
	}
	var pair []json.RawMessage
	if err := json.Unmarshal(raw, &pair); err != nil || len(pair) != 2 {
		return nil, "", fmt.Errorf("field %s is not a many2one value", key)
	}
	id, err := decodeInt64(pair[0])
	if err != nil {
		return nil, "", fmt.Errorf("field %s id: %w", key, err)
	}
	name, err := decodeString(pair[1])
	if err != nil {
		return nil, "", fmt.Errorf("field %s name: %w", key, err)
	}
	return &id, name, nil
}
