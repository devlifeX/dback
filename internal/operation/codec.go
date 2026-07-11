package operation

import (
	"encoding/json"
	"fmt"
)

func DefaultParams(kind Kind) (Params, error) {
	switch kind {
	case KindBackupDB:
		return BackupDBParams{}, nil
	case KindBackupFiles:
		return BackupFilesParams{}, nil
	case KindUpload:
		return UploadParams{StalePolicy: UploadStaleNewOnly}, nil
	default:
		return nil, fmt.Errorf("unsupported operation kind %q", kind)
	}
}

func DecodeParams(kind Kind, raw json.RawMessage) (Params, error) {
	if len(raw) == 0 {
		return DefaultParams(kind)
	}
	switch kind {
	case KindBackupDB:
		var p BackupDBParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return p, nil
	case KindBackupFiles:
		var p BackupFilesParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return p, nil
	case KindUpload:
		var p UploadParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		if p.StalePolicy == "" {
			p.StalePolicy = UploadStaleNewOnly
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unsupported operation kind %q", kind)
	}
}

func SpecFromAction(profileID, triggerRef string, kind Kind, raw json.RawMessage) (Spec, error) {
	params, err := DecodeParams(kind, raw)
	if err != nil {
		return Spec{}, err
	}
	spec := Spec{
		Kind:       kind,
		ProfileID:  profileID,
		TriggerRef: triggerRef,
		Params:     params,
	}
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}
