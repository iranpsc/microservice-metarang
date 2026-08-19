package handler

import (
	"net/http"

	pb "metarang/shared/pb/auth"
)

// Test-only exports for private helpers (external test package under tests/auth-service).

func WriteJSONForTest(w http.ResponseWriter, status int, data interface{}, skipWrap ...bool) {
	writeJSON(w, status, data, skipWrap...)
}

func WriteErrorForTest(w http.ResponseWriter, status int, message string) {
	writeError(w, status, message)
}

func WriteGRPCErrorWithLocaleForTest(w http.ResponseWriter, err error, locale string) {
	writeGRPCErrorWithLocale(w, err, locale)
}

func DecodeRequestForTest(r *http.Request, v interface{}) error {
	return decodeRequest(r, v)
}

func DecodeRequestBodyForTest(r *http.Request, v interface{}) error {
	return decodeRequestBody(r, v)
}

func GetClientIPForTest(r *http.Request) string {
	return getClientIP(r)
}

func ExtractIDFromPathForTest(path, prefix string) string {
	return extractIDFromPath(path, prefix)
}

func BuildSimplePaginationLinksForTest(r *http.Request, currentPage int32, hasMore bool) map[string]interface{} {
	return buildSimplePaginationLinks(r, currentPage, hasMore)
}

func PublicBaseURLForTest(r *http.Request) string {
	return publicBaseURL(r)
}

func GetProjectLocaleForTest() string {
	return getProjectLocale()
}

func MapServiceErrorToValidationFieldsForTest(err error, locale string) (map[string]string, bool) {
	return mapServiceErrorToValidationFields(err, locale)
}

func ReturnValidationErrorForTest(fields map[string]string) error {
	return returnValidationError(fields)
}

func UnmarshalFlexibleStringForTest(data []byte) (string, error) {
	var f flexibleString
	if err := f.UnmarshalJSON(data); err != nil {
		return "", err
	}
	return f.String(), nil
}

func UnmarshalFlexibleInt32ForTest(data []byte) (int32, error) {
	var f flexibleInt32
	if err := f.UnmarshalJSON(data); err != nil {
		return 0, err
	}
	return f.Int32(), nil
}

func RequestHasBodyForTest(r *http.Request) bool {
	return requestHasBody(r)
}

func MergeQueryParamsForTest(r *http.Request, v interface{}) {
	mergeQueryParams(r, v)
}

func WriteProfileLimitationValidationErrorsForTest(w http.ResponseWriter, fieldErrors map[string]string, locale string) {
	writeProfileLimitationValidationErrors(w, fieldErrors, locale)
}

func BuildCitizenProfileHTTPResponseForTest(resp *pb.CitizenProfileResponse) map[string]interface{} {
	return buildCitizenProfileHTTPResponse(resp)
}

func BuildCitizenReferralChartHTTPResponseForTest(resp *pb.CitizenReferralChartResponse, rangeType string) map[string]interface{} {
	return buildCitizenReferralChartHTTPResponse(resp, rangeType)
}

func UserListLevelToHTTPForTest(lvl *pb.Level) map[string]interface{} {
	return userListLevelToHTTP(lvl)
}

func WriteWalletGRPCErrorForTest(h *HTTPWalletHandler, w http.ResponseWriter, err error) {
	h.writeWalletGRPCError(w, err)
}
