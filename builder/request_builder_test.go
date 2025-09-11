package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"go.unistack.org/micro-client-http/v4/builder"
	pb "go.unistack.org/micro-client-http/v4/builder/proto"
)

func TestNewRequestBuilder(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		method    string
		bodyOpt   string
		msg       proto.Message
		wantError bool
	}{
		{
			name:      "empty path",
			path:      "",
			method:    "GET",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: true,
		},
		{
			name:      "invalid method",
			path:      "/v1/users",
			method:    "INVALID",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: true,
		},
		{
			name:      "nil msg",
			path:      "/v1/users",
			method:    "POST",
			bodyOpt:   "*",
			msg:       nil,
			wantError: true,
		},
		{
			name:      "GET without body",
			path:      "/v1/users",
			method:    "GET",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "GET with body",
			path:      "/v1/users",
			method:    "GET",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: true,
		},
		{
			name:      "DELETE without body",
			path:      "/v1/users/42",
			method:    "DELETE",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "DELETE with body",
			path:      "/v1/users/42",
			method:    "DELETE",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: true,
		},
		{
			name:      "POST with body",
			path:      "/v1/users",
			method:    "POST",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "POST without body",
			path:      "/v1/users",
			method:    "POST",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "PUT with body",
			path:      "/v1/users/42",
			method:    "PUT",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "PUT without body",
			path:      "/v1/users/42",
			method:    "PUT",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "PATCH with body",
			path:      "/v1/users/42",
			method:    "PATCH",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "PATCH without body",
			path:      "/v1/users/42",
			method:    "PATCH",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "HEAD without body",
			path:      "/v1/users/42",
			method:    "HEAD",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "HEAD with body",
			path:      "/v1/users/42",
			method:    "HEAD",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: true,
		},
		{
			name:      "OPTIONS without body",
			path:      "/v1/users/42",
			method:    "OPTIONS",
			bodyOpt:   "",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
		{
			name:      "OPTIONS with body",
			path:      "/v1/users/42",
			method:    "OPTIONS",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: true,
		},
		{
			name:      "lowercase method still valid",
			path:      "/v1/users",
			method:    "post",
			bodyOpt:   "*",
			msg:       &pb.TestRequestBuilder{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := builder.NewRequestBuilder(tt.path, tt.method, tt.bodyOpt, tt.msg)

			if tt.wantError {
				require.Error(t, err)
				require.Nil(t, rb)
			} else {
				require.NoError(t, err)
				require.NotNil(t, rb)
			}
		})
	}
}

func TestRequestBuilder_PathOnly(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		method       string
		msg          func() proto.Message
		expectedPath string
		expectedMsg  func() proto.Message
		wantError    bool
	}{
		{
			name:   "primitive case",
			path:   "/v1/users/{user_id}/orders/{order_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PrimitiveCase
				return &Msg{
					UserId:  "42",
					OrderId: 123,
				}
			},
			expectedPath: "/v1/users/42/orders/123",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PrimitiveCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "nested case",
			path:   "/v1/users/{user.id}/orders/{order.id}/products/{order.product.id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_NestedCase
				type User = pb.Test_PathOnly_NestedCase_User
				type Order = pb.Test_PathOnly_NestedCase_Order
				type Product = pb.Test_PathOnly_NestedCase_Order_Product
				return &Msg{
					User: &User{Id: "42"},
					Order: &Order{
						Id:      123,
						Product: &Product{Id: 456},
					},
				}
			},
			expectedPath: "/v1/users/42/orders/123/products/456",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_NestedCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "multiply case",
			path:   "/v1/users/{user_id}/orders/{order.id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_MultipleCase
				type Order = pb.Test_PathOnly_MultipleCase_Order
				return &Msg{
					UserId: "42",
					Order: &Order{
						Id: "123",
					},
				}
			},
			expectedPath: "/v1/users/42/orders/123",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_MultipleCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "not found case",
			path:   "/v1/users/{userId}/orders/{order_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PrimitiveCase
				return &Msg{
					UserId:  "42",
					OrderId: 123,
				}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "zero-value case",
			path:   "/v1/users/{user_id}/orders/{order_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PrimitiveCase
				return &Msg{UserId: "42"}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "repeated case",
			path:   "/v1/users/{user_id}/orders/{order_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_RepeatedCase
				return &Msg{
					UserId:  []string{"42"},
					OrderId: 123,
				}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "non-primitive message case",
			path:   "/v1/users/{user_id}/orders/{order_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_NonPrimitiveMessageCase
				type User = pb.Test_PathOnly_NonPrimitiveMessageCase_User
				return &Msg{
					UserId:  &User{Id: "42"},
					OrderId: 123,
				}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "non-primitive map case",
			path:   "/v1/users/{user_id}/orders/{order_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_NonPrimitiveMapCase
				return &Msg{
					UserId:  map[string]string{"": ""},
					OrderId: 123,
				}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "custom verb case",
			path:   "/v1/users/{user_id}:get",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PrimitiveCase
				return &Msg{UserId: "42"}
			},
			expectedPath: "/v1/users/42:get",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PrimitiveCase
				return &Msg{}
			},
			wantError: false,
		},
		// pattern cases
		{
			name:   "pattern case -> *",
			path:   "/v1/users/{pattern=*}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{Pattern: "42"}
			},
			expectedPath: "/v1/users/42",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "pattern case -> * (invalid value)",
			path:   "/v1/users/{pattern=*}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{Pattern: "a/b/c"}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "pattern case -> * (empty value)",
			path:   "/v1/users/{pattern=*}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "pattern case -> **",
			path:   "/v1/users/{pattern=**}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{Pattern: "a/b/c"}
			},
			expectedPath: "/v1/users/a/b/c",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "pattern case -> ** (empty value)",
			path:   "/v1/users/{pattern=**}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{}
			},
			expectedPath: "/v1/users/",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "pattern case -> composite pattern",
			path:   "/v1/users/{pattern=*/orders/*}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{Pattern: "42/orders/123"}
			},
			expectedPath: "/v1/users/42/orders/123",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "pattern case -> composite pattern (with extra segment)",
			path:   "/v1/users/{pattern=*/orders/*}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_PatternCase
				return &Msg{Pattern: "42/orders/123/456"}
			},
			expectedPath: "",
			expectedMsg: func() proto.Message {
				return nil
			},
			wantError: true,
		},
		{
			name:   "pattern case -> ** (composite segments)",
			path:   "/v1/users/{pattern=**}/orders/{order_id}/products/{product_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_CompositePatternCase
				return &Msg{
					Pattern:   "a/b/c",
					OrderId:   "123",
					ProductId: "456",
				}
			},
			expectedPath: "/v1/users/a/b/c/orders/123/products/456",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_CompositePatternCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "pattern case -> ** (with empty value and composite segments)",
			path:   "/v1/users/{pattern=**}/orders/{order_id}/products/{product_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_CompositePatternCase
				return &Msg{
					Pattern:   "",
					OrderId:   "123",
					ProductId: "456",
				}
			},
			expectedPath: "/v1/users//orders/123/products/456",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_CompositePatternCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "pattern case -> composite pattern (multiple consecutive variables)",
			path:   "/v1/{pattern}/{order_id}/{product_id}",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_PathOnly_CompositePatternCase
				return &Msg{
					Pattern:   "123",
					OrderId:   "456",
					ProductId: "789",
				}
			},
			expectedPath: "/v1/123/456/789",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_PathOnly_CompositePatternCase
				return &Msg{}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := builder.NewRequestBuilder(tt.path, tt.method, "", tt.msg())
			require.NoError(t, err)

			path, newMsg, err := rb.Build()
			if tt.wantError {
				require.Error(t, err)
				require.Empty(t, path)
				require.Nil(t, newMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedPath, path)
				require.True(t, proto.Equal(tt.expectedMsg(), newMsg))
			}
		})
	}
}

func TestRequestBuilder_QueryOnly(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		method       string
		msg          func() proto.Message
		expectedPath string
		expectedMsg  func() proto.Message
		wantError    bool
	}{
		{
			name:   "primitive case",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_PrimitiveCase
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Flag:    true,
				}
			},
			expectedPath: "/v1/users?flag=true&order_id=123&user_id=42",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_PrimitiveCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "primitive case (with empty fields)",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_PrimitiveCase
				return &Msg{UserId: "42"}
			},
			expectedPath: "/v1/users?user_id=42",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_PrimitiveCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "repeated case",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_RepeatedCase
				return &Msg{
					Strings:  []string{"foo", "bar"},
					Integers: []int64{1, 2, 3},
				}
			},
			expectedPath: "/v1/users?integers=1&integers=2&integers=3&strings=foo&strings=bar",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_RepeatedCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "nested message case",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_NestedMessageCase
				type Filter = pb.Test_QueryOnly_NestedMessageCase_Filter
				type SubFilter = pb.Test_QueryOnly_NestedMessageCase_Filter_SubFilter
				return &Msg{
					UserId: "42",
					Filter: &Filter{
						Age:  30,
						Name: "Alice",
						SubFilter: &SubFilter{
							SubAge:  20,
							SubName: "John",
						},
					},
				}
			},
			expectedPath: "/v1/users?filter.age=30&filter.name=Alice&filter.sub_filter.sub_age=20&filter.sub_filter.sub_name=John&user_id=42",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_NestedMessageCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "nested map case",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_NestedMapCase
				type SubFilter = pb.Test_QueryOnly_NestedMapCase_SubFilter
				return &Msg{
					UserId:      "42",
					FirstFilter: map[string]string{"age": "30", "name": "Alice"},
					SecondFilter: map[string]*SubFilter{
						"filter1": {SubAge: 20, SubName: "John"},
						"filter2": {SubAge: 40, SubName: "Travolta"},
					},
				}
			},
			expectedPath: "/v1/users?first_filter.age=30&first_filter.name=Alice&second_filter.filter1.sub_age=20&second_filter.filter1.sub_name=John&second_filter.filter2.sub_age=40&second_filter.filter2.sub_name=Travolta&user_id=42",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_NestedMapCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "multiple case",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_MultipleCase
				type Filter = pb.Test_QueryOnly_MultipleCase_Filter
				type SubFilter = pb.Test_QueryOnly_MultipleCase_SubFilter
				return &Msg{
					UserId:  "42",
					Strings: []string{"foo", "bar"},
					FirstFilter: &Filter{
						Age: 30,
						SubFilter: &SubFilter{
							SubAge: 20,
						},
					},
					SecondFilter: map[string]*SubFilter{
						"filter1": {SubAge: 20},
						"filter2": {SubAge: 40},
					},
				}
			},
			expectedPath: "/v1/users?first_filter.age=30&first_filter.sub_filter.sub_age=20&second_filter.filter1.sub_age=20&second_filter.filter2.sub_age=40&strings=foo&strings=bar&user_id=42",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_MultipleCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:   "repeated message case",
			path:   "/v1/users",
			method: "GET",
			msg: func() proto.Message {
				type Msg = pb.Test_QueryOnly_RepeatedMessageCase
				type Filter = pb.Test_QueryOnly_RepeatedMessageCase_Filter
				return &Msg{Filters: []*Filter{{Age: 20}, {Age: 30}, {Age: 40}}}
			},
			expectedPath: "",
			expectedMsg:  nil,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := builder.NewRequestBuilder(tt.path, tt.method, "", tt.msg())
			require.NoError(t, err)

			path, newMsg, err := rb.Build()
			if tt.wantError {
				require.Error(t, err)
				require.Empty(t, path)
				require.Nil(t, newMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedPath, path)
				require.True(t, proto.Equal(tt.expectedMsg(), newMsg))
			}
		})
	}
}

func TestRequestBuilder_BodyOnly(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		method       string
		bodyOpt      string
		msg          func() proto.Message
		expectedPath string
		expectedMsg  func() proto.Message
		wantError    bool
	}{
		{
			name:    "primitive case: full body",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Flag:    true,
					Strings: []string{"foo", "bar"},
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Flag:    true,
					Strings: []string{"foo", "bar"},
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			wantError: false,
		},
		{
			name:    "primitive case: specified primitive field",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "user_id",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Flag:    true,
					Strings: []string{"foo", "bar"},
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users?flag=true&order_id=123&product.id=product_id&product.name=product_name&strings=foo&strings=bar",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				return &Msg{
					UserId: "42",
				}
			},
			wantError: false,
		},
		{
			name:    "primitive case: specified non-primitive field",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "product",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Flag:    true,
					Strings: []string{"foo", "bar"},
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users?flag=true&order_id=123&strings=foo&strings=bar&user_id=42",
			expectedMsg: func() proto.Message {
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Product{
					Id:   "product_id",
					Name: "product_name",
				}
			},
			wantError: false,
		},
		{
			name:    "primitive case: empty fields",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				return &Msg{
					UserId: "42",
					Flag:   true,
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 0,
					Flag:    true,
					Strings: []string{},
					Product: &Product{},
				}
			},
			wantError: false,
		},
		{
			name:    "primitive case: empty all fields",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				return &Msg{}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				type Product = pb.Test_BodyOnly_PrimitiveCase_Product
				return &Msg{
					UserId:  "",
					OrderId: 0,
					Flag:    false,
					Strings: []string{},
					Product: &Product{},
				}
			},
			wantError: false,
		},
		{
			name:    "nested case: full body",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_NestedCase
				type Filter = pb.Test_BodyOnly_NestedCase_Filter
				type SubFilter = pb.Test_BodyOnly_NestedCase_Filter_SubFilter
				return &Msg{
					UserId: "42",
					FirstFilter: &Filter{
						Age:  30,
						Name: "Alice",
						SubFilter: &SubFilter{
							SubAge:  40,
							SubName: "John",
						},
					},
					SecondFilter: &Filter{
						Age:  50,
						Name: "Alex",
						SubFilter: &SubFilter{
							SubAge:  60,
							SubName: "Mike",
						},
					},
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_NestedCase
				type Filter = pb.Test_BodyOnly_NestedCase_Filter
				type SubFilter = pb.Test_BodyOnly_NestedCase_Filter_SubFilter
				return &Msg{
					UserId: "42",
					FirstFilter: &Filter{
						Age:  30,
						Name: "Alice",
						SubFilter: &SubFilter{
							SubAge:  40,
							SubName: "John",
						},
					},
					SecondFilter: &Filter{
						Age:  50,
						Name: "Alex",
						SubFilter: &SubFilter{
							SubAge:  60,
							SubName: "Mike",
						},
					},
				}
			},
			wantError: false,
		},
		{
			name:    "nested case: specified non-primitive field",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "second_filter",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_NestedCase
				type Filter = pb.Test_BodyOnly_NestedCase_Filter
				type SubFilter = pb.Test_BodyOnly_NestedCase_Filter_SubFilter
				return &Msg{
					UserId: "42",
					FirstFilter: &Filter{
						Age:  30,
						Name: "Alice",
						SubFilter: &SubFilter{
							SubAge:  40,
							SubName: "John",
						},
					},
					SecondFilter: &Filter{
						Age:  50,
						Name: "Alex",
						SubFilter: &SubFilter{
							SubAge:  60,
							SubName: "Mike",
						},
					},
				}
			},
			expectedPath: "/v1/users?first_filter.age=30&first_filter.name=Alice&first_filter.sub_filter.sub_age=40&first_filter.sub_filter.sub_name=John&user_id=42",
			expectedMsg: func() proto.Message {
				type Filter = pb.Test_BodyOnly_NestedCase_Filter
				type SubFilter = pb.Test_BodyOnly_NestedCase_Filter_SubFilter
				return &Filter{
					Age:  50,
					Name: "Alex",
					SubFilter: &SubFilter{
						SubAge:  60,
						SubName: "Mike",
					},
				}
			},
			wantError: false,
		},
		{
			name:    "repeated message case",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_RepeatedMessageCase
				type Product = pb.Test_BodyOnly_RepeatedMessageCase_Product
				return &Msg{
					UserId: "42",
					Products: []*Product{
						{Id: "product_id_1", Name: "product_name_1"},
						{Id: "product_id_2", Name: "product_name_2"},
					},
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_RepeatedMessageCase
				type Product = pb.Test_BodyOnly_RepeatedMessageCase_Product
				return &Msg{
					UserId: "42",
					Products: []*Product{
						{Id: "product_id_1", Name: "product_name_1"},
						{Id: "product_id_2", Name: "product_name_2"},
					},
				}
			},
			wantError: false,
		},
		{
			name:    "repeated message case (empty)",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_RepeatedMessageCase
				type Product = pb.Test_BodyOnly_RepeatedMessageCase_Product
				return &Msg{
					UserId:   "42",
					Products: []*Product{},
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_RepeatedMessageCase
				type Product = pb.Test_BodyOnly_RepeatedMessageCase_Product
				return &Msg{
					UserId:   "42",
					Products: []*Product{},
				}
			},
			wantError: false,
		},
		{
			name:    "primitive and non-primitive map case",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MapCase
				type SubFilter = pb.Test_BodyOnly_MapCase_SubFilter
				return &Msg{
					FirstFilter: map[string]string{"age": "50", "name": "Alex"},
					SecondFilter: map[string]*SubFilter{
						"second_filter_1": {SubAge: 30, SubName: "Alice"},
						"second_filter_2": {SubAge: 40, SubName: "John"},
					},
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MapCase
				type SubFilter = pb.Test_BodyOnly_MapCase_SubFilter
				return &Msg{
					FirstFilter: map[string]string{"age": "50", "name": "Alex"},
					SecondFilter: map[string]*SubFilter{
						"second_filter_1": {SubAge: 30, SubName: "Alice"},
						"second_filter_2": {SubAge: 40, SubName: "John"},
					},
				}
			},
			wantError: false,
		},
		{
			name:    "primitive and non-primitive map case (empty)",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MapCase
				return &Msg{}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MapCase
				type SubFilter = pb.Test_BodyOnly_MapCase_SubFilter
				return &Msg{
					FirstFilter:  map[string]string{},
					SecondFilter: map[string]*SubFilter{},
				}
			},
			wantError: false,
		},
		{
			name:    "multiple case",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MultipleCase
				type SubFilter = pb.Test_BodyOnly_MultipleCase_SubFilter
				return &Msg{
					UserId: "42",
					FirstFilter: []*SubFilter{
						{SubAge: 30, SubName: "Alice"},
						{SubAge: 40, SubName: "John"},
					},
					SecondFilter: map[string]*SubFilter{
						"second_filter_1": {SubAge: 50, SubName: "Alex"},
						"second_filter_2": {SubAge: 60, SubName: "Max"},
					},
					ThirdFilter: &SubFilter{SubAge: 70, SubName: "Ricardo"},
				}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MultipleCase
				type SubFilter = pb.Test_BodyOnly_MultipleCase_SubFilter
				return &Msg{
					UserId: "42",
					FirstFilter: []*SubFilter{
						{SubAge: 30, SubName: "Alice"},
						{SubAge: 40, SubName: "John"},
					},
					SecondFilter: map[string]*SubFilter{
						"second_filter_1": {SubAge: 50, SubName: "Alex"},
						"second_filter_2": {SubAge: 60, SubName: "Max"},
					},
					ThirdFilter: &SubFilter{SubAge: 70, SubName: "Ricardo"},
				}
			},
			wantError: false,
		},
		{
			name:    "multiple case (empty)",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MultipleCase
				return &Msg{}
			},
			expectedPath: "/v1/users",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_MultipleCase
				type SubFilter = pb.Test_BodyOnly_MultipleCase_SubFilter
				return &Msg{
					UserId:       "",
					FirstFilter:  []*SubFilter{},
					SecondFilter: map[string]*SubFilter{},
					ThirdFilter:  &SubFilter{},
				}
			},
			wantError: false,
		},
		{
			name:    "nonexistent body field",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "nonexistent_field",
			msg: func() proto.Message {
				type Msg = pb.Test_BodyOnly_PrimitiveCase
				return &Msg{}
			},
			expectedPath: "",
			expectedMsg:  nil,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := builder.NewRequestBuilder(tt.path, tt.method, tt.bodyOpt, tt.msg())
			require.NoError(t, err)

			path, newMsg, err := rb.Build()
			if tt.wantError {
				require.Error(t, err)
				require.Empty(t, path)
				require.Nil(t, newMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedPath, path)
				require.True(t, proto.Equal(tt.expectedMsg(), newMsg))
			}
		})
	}
}

func TestRequestBuilder_Mixed(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		method       string
		bodyOpt      string
		msg          func() proto.Message
		expectedPath string
		expectedMsg  func() proto.Message
		wantError    bool
	}{
		{
			name:    "path + query",
			path:    "/v1/users/{user_id}/orders/{order_id}",
			method:  "POST",
			bodyOpt: "",
			msg: func() proto.Message {
				type Msg = pb.Test_Mixed_PrimitiveCase
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users/42/orders/123?product.id=product_id&product.name=product_name",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_Mixed_PrimitiveCase
				return &Msg{}
			},
			wantError: false,
		},
		{
			name:    "path + body",
			path:    "/v1/users/{user_id}/orders/{order_id}",
			method:  "POST",
			bodyOpt: "*",
			msg: func() proto.Message {
				type Msg = pb.Test_Mixed_PrimitiveCase
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users/42/orders/123",
			expectedMsg: func() proto.Message {
				type Msg = pb.Test_Mixed_PrimitiveCase
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Msg{
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			wantError: false,
		},
		{
			name:    "path + body + query",
			path:    "/v1/users/{user_id}",
			method:  "POST",
			bodyOpt: "product",
			msg: func() proto.Message {
				type Msg = pb.Test_Mixed_PrimitiveCase
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users/42?order_id=123",
			expectedMsg: func() proto.Message {
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Product{
					Id:   "product_id",
					Name: "product_name",
				}
			},
			wantError: false,
		},
		{
			name:    "query + body",
			path:    "/v1/users",
			method:  "POST",
			bodyOpt: "product",
			msg: func() proto.Message {
				type Msg = pb.Test_Mixed_PrimitiveCase
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Msg{
					UserId:  "42",
					OrderId: 123,
					Product: &Product{
						Id:   "product_id",
						Name: "product_name",
					},
				}
			},
			expectedPath: "/v1/users?order_id=123&user_id=42",
			expectedMsg: func() proto.Message {
				type Product = pb.Test_Mixed_PrimitiveCase_Product
				return &Product{
					Id:   "product_id",
					Name: "product_name",
				}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb, err := builder.NewRequestBuilder(tt.path, tt.method, tt.bodyOpt, tt.msg())
			require.NoError(t, err)

			path, newMsg, err := rb.Build()
			if tt.wantError {
				require.Error(t, err)
				require.Empty(t, path)
				require.Nil(t, newMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedPath, path)
				require.True(t, proto.Equal(tt.expectedMsg(), newMsg))
			}
		})
	}
}
