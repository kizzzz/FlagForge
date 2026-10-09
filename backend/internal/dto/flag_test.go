package dto

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"flagforge/internal/apperror"
	"github.com/stretchr/testify/require"
)

func TestPagingRejectsAmbiguousOrUnsafeInputs(t *testing.T) {
	for _, query := range []string{"page=1&page=2", "page_size=20&page_size=50", "page=1.1", "page=-1", "page_size=-1", "page=9223372036854775808", "page=" + strconv.Itoa(int(^uint(0)>>1)) + "&page_size=100"} {
		t.Run(query, func(t *testing.T) {
			values, err := url.ParseQuery(query)
			require.NoError(t, err)
			_, err = ParsePageQuery(values)
			require.Error(t, err)
			var business *apperror.Error
			require.True(t, errors.As(err, &business))
			require.Equal(t, apperror.ValidationError, business.Code)
		})
	}
}
func TestFlagValidationBoundaries(t *testing.T) {
	for _, key := range []string{"A1", "a b", "中文", "1a", "a", "a" + strings.Repeat("b", 64)} {
		require.Error(t, ValidateKey(key), key)
	}
	require.NoError(t, ValidateKey("a"+strings.Repeat("b", 63)))
	r, err := DecodeCreateFlag([]byte(`{"name":"\u3000` + strings.Repeat("中", 100) + `\u3000","key":"ok","environment":"staging","default_value":true,"enabled":false}`))
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("中", 100), r.Name)
	require.Equal(t, false, *r.Enabled)
	for _, body := range []string{`{"name":"","key":"ok","environment":"staging","default_value":true}`, `{"name":"x","key":"ok","environment":"","default_value":true}`, `{"name":"x","key":"ok","environment":"staging","default_value":true,"enabled":null}`} {
		_, err = DecodeCreateFlag([]byte(body))
		require.Error(t, err)
	}
	for _, body := range []string{`{"k\u0065y":null}`, `{"key":null,"name":"x"}`, `{"environment":false}`, `{"key":"first","key":"second"}`} {
		_, err = DecodeUpdateFlag([]byte(body))
		var business *apperror.Error
		require.True(t, errors.As(err, &business))
		require.Equal(t, apperror.ImmutableField, business.Code)
	}
	_, err = DecodeCreateFlag([]byte(`{"name":"x",}`))
	var business *apperror.Error
	require.True(t, errors.As(err, &business))
	require.Equal(t, apperror.InvalidJSON, business.Code)
	require.Contains(t, business.Message, "byte")
}
