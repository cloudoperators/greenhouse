// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scim

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

type mockResponse struct {
	statusCode int
	body       string
}

func emptyResponseBodyMockFn() mockResponse {
	return mockResponse{
		statusCode: http.StatusOK,
		body:       emptyResponseBodyMock,
	}
}

func existingGroupResponseBodyMockFn() mockResponse {
	return mockResponse{
		statusCode: http.StatusOK,
		body:       groupResponseBodyMock,
	}
}

func errorResponseBodyMockFn() mockResponse {
	return mockResponse{
		statusCode: http.StatusBadRequest,
		body:       errorResponseBodyMock,
	}
}

func userResponseBodyMockFn() mockResponse {
	return mockResponse{
		statusCode: http.StatusOK,
		body:       userResponseBodyMockTwo,
	}
}

func firstUserPaginatedResponseFn() mockResponse {
	return mockResponse{
		statusCode: http.StatusOK,
		body: `{
            "totalResults": 2,
            "Resources": [
                {"id": "user1"},
                {"id": "user2"}
            ],
            "nextId": "second-page"
        }`,
	}
}

func secondUserPaginatedResponseFn() mockResponse {
	return mockResponse{
		statusCode: http.StatusOK,
		body: `{
			"totalResults": 1,
			"Resources": [
				{"id": "user3"}
			],
			"nextId": "end"
		}`,
	}
}

const (
	baseURLPath           = "/scim"
	errorResponseBodyMock = `{"schemas":["urn:ietf:params:scim:api:messages:2.0:Error"],"status": "400","detail": "Invalid startId"}`
	emptyResponseBodyMock = `{"totalResults": 0,"itemsPerPage": 100,"startIndex" : 1, "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"]}`

	groupResponseBodyMock = `{
		"Resources" : [{
		  "id": "123",
		  "meta": {
			"created": "2022-02-07T13:44:57Z",
			"lastModified": "2023-03-03T09:01:28Z",
			"location": "https://some-location",
			"version": "3ed780be-888b-4412-9915-661d31e30457",
			"resourceType": "Group"
		  },
		  "displayName": "SOME_IDP_GROUP_NAME"
		}],
		"totalResults": 1,
		"itemsPerPage": 100,
		"startIndex": 1,
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"]
	  }`

	otherGroupResponseBodyMock = `{
		"Resources" : [{
		  "id": "456",
		  "meta": {
			"created": "2022-02-07T13:44:57Z",
			"lastModified": "2023-03-03T09:01:28Z",
			"location": "https://other-location",
			"version": "3ed780be-888b-4412-9915-661d31e30457",
			"resourceType": "Group"
		  },
		  "displayName": "ANOTHER_IDP_GROUP"
		}],
		"totalResults": 1,
		"itemsPerPage": 100,
		"startIndex": 1,
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"]
	  }`

	userResponseBodyMockTwo = `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
		"totalResults": 2,
		"itemsPerPage": 100,
		"startIndex": 1,
		"Resources": [ {
			"id": "12345",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://some-location",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I12345",
			"name": {
		  		"familyName": "Doe",
		  		"givenName": "John"
			},
			"displayName": "John Doe",
			"userType": "employee",
			"active": true,
			"emails": [{"value": "john.doe@example.com", "primary": true}]
		},
		{
			"id": "23456",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://some-location",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I23456",
			"name": {
		  		"familyName": "Doe",
		  		"givenName": "Jane"
			},
			"displayName": "Jane Doe",
			"userType": "employee",
			"active": true,
			"emails": [{"value": "jane.doe@example.com", "primary": true}]
		}]
	}`

	userResponseBodyMockThree = `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
		"totalResults": 2,
		"itemsPerPage": 100,
		"startIndex": 1,
		"Resources": [ {
			"id": "12345",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://lost-ark",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I12345",
			"name": {
		  		"familyName": "Jones",
		  		"givenName": "Indiana"
			},
			"displayName": "Indiana Jones",
			"userType": "employee",
			"active": true,
			"emails": [{"value": "indiana.jones@lost-ark.com", "primary": true}]
		},
		{
			"id": "23456",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://some-location",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I23456",
			"name": {
		  		"familyName": "Croft",
		  		"givenName": "Lara"
			},
			"displayName": "Lara Croft",
			"userType": "employee",
			"active": true,
			"emails": [{"value": "lara.croft@tomb-raider.com", "primary": true}]
		},
		{
			"id": "34567",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://death-star",
		  		"version": "345",
		  		"resourceType": "User"
			},
			"userName": "I34567",
			"name": {
		  		"familyName": "Vader",
		  		"givenName": "Darth"
			},
			"displayName": "Darth Vader",
			"userType": "employee",
			"active": true,
			"emails": [{"value": "darth.vader@death-star.com", "primary": true}]
		}]
	}`

	//nolint:unused
	inactiveUserResponseBodyMock = `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
		"totalResults": 1,
		"itemsPerPage": 100,
		"startIndex": 1,
		"Resources": [{
			"id": "12345",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://some-location",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I12345",
			"name": {
		  		"familyName": "Doe",
		  		"givenName": "John"
			},
			"displayName": "John Doe",
			"userType": "employee",
			"active": true,
			"emails": [ { "value": "john.doe@example.com", "primary": true } ]
		},
		{
			"id": "23456",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://some-location",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I23456",
			"name": {
		  		"familyName": "Doe",
		  		"givenName": "Jane"
			},
			"displayName": "Jane Doe",
			"userType": "employee",
			"active": true,
			"emails": [{"value": "jane.doe@example.com", "primary": true}]
		},
		{
			"id": "78901",
			"meta": {
		  		"created": "2020-09-02T01:19:13Z",
		  		"lastModified": "2022-11-08T00:51:04Z",
		  		"location": "https://some-location",
		  		"version": "123",
		  		"resourceType": "User"
			},
			"userName": "I7654",
			"name": {
		  		"familyName": "Inactive",
		  		"givenName": "John"
			},
			"displayName": "John Inactive",
			"userType": "employee",
			"active": false,
			"emails": [{"value": "john.inactive@example.com", "primary": true}],
	  	}]
	}`
)

func setup() (*httptest.Server, *http.ServeMux) {
	mux := http.NewServeMux()

	// We want to ensure that mocks catch mistakes where the endpoint URL is
	// specified as absolute rather than relative.
	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		_, _ = fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		_, _ = fmt.Fprintln(os.Stderr)
		_, _ = fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		_, _ = fmt.Fprintln(os.Stderr)
		_, _ = fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)
	return server, mux
}

func ReturnDefaultGroupResponseMockServer() *httptest.Server {
	server, mux := setup()
	mux.HandleFunc("/Groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/scim+json")
		switch r.URL.RawQuery {
		case "excludedAttributes=members&filter=displayName+eq+%22SOME_IDP_GROUP_NAME%22":
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprint(w, groupResponseBodyMock)
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		case "excludedAttributes=members&filter=displayName+eq+%22ANOTHER_IDP_GROUP%22":
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprint(w, otherGroupResponseBodyMock)
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		case "excludedAttributes=members&filter=displayName+eq+%22NON_EXISTING_GROUP_NAME%22":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(emptyResponseBodyMock))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{}`))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		}
	})
	return server
}

func ReturnUserResponseMockServer() *httptest.Server {
	server, mux := setup()
	mux.HandleFunc("/Users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/scim+json")
		switch r.URL.RawQuery {
		case "attributes=name%2Cemails%2CdisplayName%2Cactive&filter=groups.display+eq+%22SOME_IDP_GROUP_NAME%22&startId=initial":
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprint(w, userResponseBodyMockTwo)
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		case "attributes=name%2Cemails%2CdisplayName%2Cactive&filter=groups.display+eq+%22SOME_OTHER_IDP_GROUP_NAME%22&startId=initial":
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprint(w, userResponseBodyMockThree)
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		case "filter=displayName+eq+%22NON_EXISTING_GROUP_NAME%22":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(emptyResponseBodyMock))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{}`))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		}
	})
	return server
}
