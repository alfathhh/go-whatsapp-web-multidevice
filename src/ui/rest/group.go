package rest

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"

	domainGroup "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/group"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
)

type Group struct {
	Service domainGroup.IGroupUsecase
}

func InitRestGroup(app fiber.Router, service domainGroup.IGroupUsecase) Group {
	rest := Group{Service: service}
	app.Post("/group", rest.CreateGroup)
	app.Post("/group/participants/import", rest.ImportParticipants)
	app.Post("/group/join-with-link", rest.JoinGroupWithLink)
	app.Get("/group/info-from-link", rest.GetGroupInfoFromLink)
	app.Get("/group/info", rest.GroupInfo)
	app.Post("/group/leave", rest.LeaveGroup)
	app.Get("/group/participants", rest.ListParticipants)
	app.Get("/group/participants/export", rest.ExportParticipants)
	app.Post("/group/participants", rest.AddParticipants)
	app.Post("/group/participants/remove", rest.DeleteParticipants)
	app.Post("/group/participants/promote", rest.PromoteParticipants)
	app.Post("/group/participants/demote", rest.DemoteParticipants)
	app.Get("/group/participant-requests", rest.ListParticipantRequests)
	app.Post("/group/participant-requests/approve", rest.ApproveParticipantRequests)
	app.Post("/group/participant-requests/reject", rest.RejectParticipantRequests)
	app.Post("/group/photo", rest.SetGroupPhoto)
	app.Post("/group/name", rest.SetGroupName)
	app.Post("/group/locked", rest.SetGroupLocked)
	app.Post("/group/announce", rest.SetGroupAnnounce)
	app.Post("/group/topic", rest.SetGroupTopic)
	app.Get("/group/invite-link", rest.GetGroupInviteLink)
	return rest
}

func (controller *Group) JoinGroupWithLink(c *fiber.Ctx) error {
	var request domainGroup.JoinGroupWithLinkRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	response, err := controller.Service.JoinGroupWithLink(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success joined group",
		Results: map[string]string{
			"group_id": response,
		},
	})
}

func (controller *Group) GetGroupInfoFromLink(c *fiber.Ctx) error {
	var request domainGroup.GetGroupInfoFromLinkRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	response, err := controller.Service.GetGroupInfoFromLink(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success get group info from link",
		Results: response,
	})
}

func (controller *Group) LeaveGroup(c *fiber.Ctx) error {
	var request domainGroup.LeaveGroupRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	err = controller.Service.LeaveGroup(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success leave group",
	})
}

func (controller *Group) CreateGroup(c *fiber.Ctx) error {
	var request domainGroup.CreateGroupRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	groupID, err := controller.Service.CreateGroup(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Success created group with id %s", groupID),
		Results: map[string]string{
			"group_id": groupID,
		},
	})
}

func (controller *Group) ListParticipants(c *fiber.Ctx) error {
	var request domainGroup.GetGroupParticipantsRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	if request.GroupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_GROUP_ID",
			Message: "Group ID cannot be empty",
		})
	}

	utils.SanitizePhone(&request.GroupID)

	result, err := controller.Service.GetGroupParticipants(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success getting group participants",
		Results: result,
	})
}

func (controller *Group) ExportParticipants(c *fiber.Ctx) error {
	var request domainGroup.GetGroupParticipantsRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	if request.GroupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_GROUP_ID",
			Message: "Group ID cannot be empty",
		})
	}

	utils.SanitizePhone(&request.GroupID)

	result, err := controller.Service.GetGroupParticipants(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	utils.PanicIfNeeded(writer.Write([]string{"participant_jid", "phone_number", "lid", "display_name", "role"}))

	for _, participant := range result.Participants {
		role := "member"
		if participant.IsSuperAdmin {
			role = "super_admin"
		} else if participant.IsAdmin {
			role = "admin"
		}

		record := []string{
			participant.JID,
			participant.PhoneNumber,
			participant.LID,
			participant.DisplayName,
			role,
		}

		utils.PanicIfNeeded(writer.Write(record))
	}

	writer.Flush()
	utils.PanicIfNeeded(writer.Error())

	fileName := fmt.Sprintf("group-%s-participants.csv", strings.ReplaceAll(result.GroupID, "@", "_"))

	c.Type("text/csv; charset=utf-8")
	c.Attachment(fileName)

	return c.Send(buffer.Bytes())
}

func (controller *Group) AddParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangeAdd, "Success add participants")
}

func (controller *Group) DeleteParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangeRemove, "Success delete participants")
}

func (controller *Group) PromoteParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangePromote, "Success promote participants")
}

func (controller *Group) DemoteParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangeDemote, "Success demote participants")
}

func (controller *Group) ListParticipantRequests(c *fiber.Ctx) error {
	var request domainGroup.GetGroupRequestParticipantsRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	if request.GroupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_GROUP_ID",
			Message: "Group ID cannot be empty",
		})
	}

	utils.SanitizePhone(&request.GroupID)

	result, err := controller.Service.GetGroupRequestParticipants(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success getting list requested participants",
		Results: result,
	})
}

func (controller *Group) ApproveParticipantRequests(c *fiber.Ctx) error {
	return controller.handleRequestedParticipants(c, whatsmeow.ParticipantChangeApprove, "Success approve requested participants")
}

func (controller *Group) RejectParticipantRequests(c *fiber.Ctx) error {
	return controller.handleRequestedParticipants(c, whatsmeow.ParticipantChangeReject, "Success reject requested participants")
}

// Generalized participant management handler
func (controller *Group) manageParticipants(c *fiber.Ctx, action whatsmeow.ParticipantChange, successMsg string) error {
	var request domainGroup.ParticipantRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	utils.SanitizePhone(&request.GroupID)
	request.Action = action
	result, err := controller.Service.ManageParticipant(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: successMsg,
		Results: result,
	})
}

// Generalized requested participants handler
func (controller *Group) handleRequestedParticipants(c *fiber.Ctx, action whatsmeow.ParticipantRequestChange, successMsg string) error {
	var request domainGroup.GroupRequestParticipantsRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	utils.SanitizePhone(&request.GroupID)
	request.Action = action
	result, err := controller.Service.ManageGroupRequestParticipants(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: successMsg,
		Results: result,
	})
}

func (controller *Group) SetGroupPhoto(c *fiber.Ctx) error {
	var request domainGroup.SetGroupPhotoRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	file, err := c.FormFile("photo")
	if err == nil {
		logrus.Printf("INFO: Received group photo - Filename: %s, Size: %d bytes, ContentType: %s",
			file.Filename, file.Size, file.Header.Get("Content-Type"))

		// Basic validation only - processing will be done in usecase
		if err := utils.ValidateGroupPhotoFormat(file); err != nil {
			logrus.Printf("ERROR: Group photo validation failed - %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
				Status:  400,
				Code:    "INVALID_IMAGE_FORMAT",
				Message: fmt.Sprintf("Image validation failed: %v", err),
			})
		}

		request.Photo = file
	} else {
		logrus.Printf("DEBUG: No photo file provided - Error: %v", err)
	}

	pictureID, err := controller.Service.SetGroupPhoto(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	if err != nil {
		logrus.Printf("ERROR: WhatsApp service failed to set group photo - %v", err)
	}
	utils.PanicIfNeeded(err)

	message := "Success update group photo"
	if request.Photo == nil {
		message = "Success remove group photo"
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: message,
		Results: domainGroup.SetGroupPhotoResponse{
			PictureID: pictureID,
			Message:   message,
		},
	})
}

func (controller *Group) SetGroupName(c *fiber.Ctx) error {
	var request domainGroup.SetGroupNameRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	err = controller.Service.SetGroupName(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Success update group name to '%s'", request.Name),
	})
}

func (controller *Group) SetGroupLocked(c *fiber.Ctx) error {
	var request domainGroup.SetGroupLockedRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	err = controller.Service.SetGroupLocked(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	message := "Success set group as unlocked"
	if request.Locked {
		message = "Success set group as locked"
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: message,
	})
}

func (controller *Group) SetGroupAnnounce(c *fiber.Ctx) error {
	var request domainGroup.SetGroupAnnounceRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	err = controller.Service.SetGroupAnnounce(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	message := "Success disable announce mode"
	if request.Announce {
		message = "Success enable announce mode"
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: message,
	})
}

func (controller *Group) SetGroupTopic(c *fiber.Ctx) error {
	var request domainGroup.SetGroupTopicRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	err = controller.Service.SetGroupTopic(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	message := "Success update group topic"
	if request.Topic == "" {
		message = "Success remove group topic"
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: message,
	})
}

// GroupInfo handles the /group/info endpoint to fetch group information
func (controller *Group) GroupInfo(c *fiber.Ctx) error {
	var request domainGroup.GroupInfoRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	response, err := controller.Service.GroupInfo(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success get group info",
		Results: response.Data,
	})
}

func (controller *Group) GetGroupInviteLink(c *fiber.Ctx) error {
	var request domainGroup.GetGroupInviteLinkRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	utils.SanitizePhone(&request.GroupID)

	response, err := controller.Service.GetGroupInviteLink(whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success get group invite link",
		Results: response,
	})
}


// ImportParticipants handles POST /group/participants/import
// Menerima file Excel (.xlsx / .xls) atau CSV, membaca kolom nomor HP,
// lalu menambahkan semua nomor tersebut sebagai peserta grup.
//
// Form fields:
//   group_id  – ID grup (tanpa @g.us, akan ditambahkan otomatis jika belum ada)
//   file      – file Excel (.xlsx) atau CSV
//
// Format file:
//   - Baris pertama boleh header (akan dilewati jika isinya bukan angka)
//   - Kolom pertama berisi nomor HP internasional (misal: 6281234567890)
//   - Kolom tambahan diabaikan
func (controller *Group) ImportParticipants(c *fiber.Ctx) error {
	groupID := strings.TrimSpace(c.FormValue("group_id"))
	if groupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_GROUP_ID",
			Message: "group_id is required",
		})
	}
	utils.SanitizePhone(&groupID)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "FILE_REQUIRED",
			Message: "file is required (.xlsx or .csv)",
		})
	}

	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Status:  500,
			Code:    "FILE_OPEN_ERROR",
			Message: fmt.Sprintf("cannot open uploaded file: %v", err),
		})
	}
	defer f.Close()

	filename := strings.ToLower(fileHeader.Filename)
	var phones []string

	switch {
	case strings.HasSuffix(filename, ".csv"):
		phones, err = parseCSV(f)
	case strings.HasSuffix(filename, ".xlsx"), strings.HasSuffix(filename, ".xls"):
		phones, err = parseExcel(f)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "UNSUPPORTED_FILE_TYPE",
			Message: "only .xlsx, .xls, and .csv files are supported",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "FILE_PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse file: %v", err),
		})
	}

	if len(phones) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "NO_PARTICIPANTS",
			Message: "no valid phone numbers found in the file",
		})
	}

	logrus.Infof("ImportParticipants: group=%s total=%d phones parsed from %s", groupID, len(phones), fileHeader.Filename)

	request := domainGroup.ParticipantRequest{
		GroupID:      groupID,
		Participants: phones,
		Action:       whatsmeow.ParticipantChangeAdd,
	}

	result, err := controller.Service.ManageParticipant(
		whatsapp.ContextWithDevice(c.UserContext(), getDeviceFromCtx(c)),
		request,
	)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Processed %d participants from file", len(phones)),
		Results: result,
	})
}

// digitOnly strips all non-digit characters from a string.
var reNonDigit = regexp.MustCompile(`\D`)

func sanitizePhone(raw string) string {
	return reNonDigit.ReplaceAllString(strings.TrimSpace(raw), "")
}

// parseCSV reads phone numbers from the first column of a CSV file.
// Baris header (nilai tidak numerik murni) dilewati secara otomatis.
func parseCSV(r io.Reader) ([]string, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // allow variable columns

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var phones []string
	for _, row := range records {
		if len(row) == 0 {
			continue
		}
		phone := sanitizePhone(row[0])
		if phone == "" {
			continue
		}
		phones = append(phones, phone)
	}
	return phones, nil
}

// parseExcel reads phone numbers from the first column of the first sheet
// of an Excel file (.xlsx or .xls).
func parseExcel(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	xlsx, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot open excel: %w", err)
	}
	defer xlsx.Close()

	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel file has no sheets")
	}

	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("cannot read sheet '%s': %w", sheets[0], err)
	}

	var phones []string
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		phone := sanitizePhone(row[0])
		if phone == "" {
			continue
		}
		phones = append(phones, phone)
	}
	return phones, nil
}
