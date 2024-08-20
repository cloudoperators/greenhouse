// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scim

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
)

const (
	EmptyGroupResponseBodyMock = `{"totalResults" : 0,"itemsPerPage" : 100,"startIndex" : 1,"schemas" : [ "urn:ietf:params:scim:api:messages:2.0:ListResponse" ]}`

	GroupResponseBodyWithMembersMock = `{
		"Resources" : [ {
		  "id" : "123",
		  "meta" : {
			"created" : "2022-02-07T13:44:57Z",
			"lastModified" : "2023-03-03T09:01:28Z",
			"location" : "https://some-location",
			"version" : "3ed780be-888b-4412-9915-661d31e30457",
			"resourceType" : "Group"
		  },
		  "schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:Group", "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" ],
		  "displayName" : "SOME_IDP_GROUP_NAME",
		  "members" : [ {
			"value" : "123",
			"$ref" : "https://some-user-location",
			"type" : "User"
		  }, {
			"value" : "234",
			"$ref" : "https://some-other-user-location",
			"type" : "User"
		  }, {
			"value" : "345",
			"$ref" : "https://some-third-user-location",
			"type" : "User"
		  }],
		  "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" : {
			"name" : "SOME_IDP_GROUP_NAME",
			"additionalId" : "123",
			"description" : "SOME_IDP_GROUP_NAME"
		  }
		} ],
		"totalResults" : 1,
		"itemsPerPage" : 100,
		"startIndex" : 1,
		"schemas" : [ "urn:ietf:params:scim:api:messages:2.0:ListResponse" ]
	  }`
	MalformedGroupResponseBodyMock = `{
		"Resources" : [ {
		  "id" : "malformedGroupResponse",
		  "displayName" : "SOME_IDP_GROUP_NAME"
		} ],
		"totalResults" : 1,
		"itemsPerPage" : 100,
		"startIndex" : 1,
		"schemas" : [ "urn:ietf:params:scim:api:messages:2.0:ListResponse" ]
	  }`

	UserResponseBodyMock1 = `{
		"id" : "12345",
		"meta" : {
		  "created" : "2020-09-02T01:19:13Z",
		  "lastModified" : "2022-11-08T00:51:04Z",
		  "location" : "https://some-location",
		  "version" : "123",
		  "resourceType" : "User"
		},
		"schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:User", "urn:ietf:params:scim:schemas:extension:sap:2.0:User" ],
		"userName" : "I12345",
		"name" : {
		  "familyName" : "Doe",
		  "givenName" : "John"
		},
		"displayName" : "John Doe",
		"userType" : "employee",
		"active" : true,
		"emails" : [ {
		  "value" : "john.doe@example.com",
		  "primary" : true
		} ],
		"groups" : [ {
		  "value" : "12345",
		  "display" : "SOME_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-location"
		}, {
		  "value" : "23456",
		  "display" : "SOME_OTHER_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-other-location"
		} ],
		"urn:ietf:params:scim:schemas:extension:sap:2.0:User" : {
		  "mock" : "empty"
		}
	  }`

	MalFormedUserResponseBodyMock = `{
		"id": "malFormedUser",
		"userName" : "I12345",
		"active" : true
	  }`

	UserResponseBodyMock2 = `{
		"id" : "23456",
		"meta" : {
		  "created" : "2020-09-02T01:19:13Z",
		  "lastModified" : "2022-11-08T00:51:04Z",
		  "location" : "https://some-location",
		  "version" : "123",
		  "resourceType" : "User"
		},
		"schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:User", "urn:ietf:params:scim:schemas:extension:sap:2.0:User" ],
		"userName" : "I23456",
		"name" : {
		  "familyName" : "Doe",
		  "givenName" : "Jane"
		},
		"displayName" : "Jane Doe",
		"userType" : "employee",
		"active" : true,
		"emails" : [ {
		  "value" : "jane.doe@example.com",
		  "primary" : true
		} ],
		"groups" : [ {
		  "value" : "12345",
		  "display" : "SOME_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-location"
		}, {
		  "value" : "23456",
		  "display" : "SOME_OTHER_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-other-location"
		} ],
		"urn:ietf:params:scim:schemas:extension:sap:2.0:User" : {
		  "mock" : "empty"
		}
	  }`

	GroupResponseBodyWith3MembersAndURLMock = `{
		"Resources" : [ {
		  "id" : "123",
		  "meta" : {
			"created" : "2022-02-07T13:44:57Z",
			"lastModified" : "2023-03-03T09:01:28Z",
			"location" : "https://some-location",
			"version" : "3ed780be-888b-4412-9915-661d31e30457",
			"resourceType" : "Group"
		  },
		  "schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:Group", "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" ],
		  "displayName" : "SOME_IDP_GROUP_NAME",
		  "members" : [ {
			"value" : "123",
			"$ref" : "%s",
			"type" : "User"
		  }, {
			"value" : "234",
			"$ref" : "%s",
			"type" : "User"
		  }, {
			"value" : "345",
			"$ref" : "%s",
			"type" : "User"
		  }],
		  "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" : {
			"name" : "SOME_IDP_GROUP_NAME",
			"additionalId" : "123",
			"description" : "SOME_IDP_GROUP_NAME"
		  }
		} ],
		"totalResults" : 1,
		"itemsPerPage" : 100,
		"startIndex" : 1,
		"schemas" : [ "urn:ietf:params:scim:api:messages:2.0:ListResponse" ]
	  }`

	GroupResponseBodyWith2MembersAndURLMock = `{
		"Resources" : [ {
		  "id" : "123",
		  "meta" : {
			"created" : "2022-02-07T13:44:57Z",
			"lastModified" : "2023-03-03T09:01:28Z",
			"location" : "https://some-location",
			"version" : "3ed780be-888b-4412-9915-661d31e30457",
			"resourceType" : "Group"
		  },
		  "schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:Group", "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" ],
		  "displayName" : "SOME_IDP_GROUP_NAME",
		  "members" : [ {
			"value" : "123",
			"$ref" : "%s",
			"type" : "User"
		  }, {
			"value" : "234",
			"$ref" : "%s",
			"type" : "User"
		  }],
		  "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" : {
			"name" : "SOME_IDP_GROUP_NAME",
			"additionalId" : "123",
			"description" : "SOME_IDP_GROUP_NAME"
		  }
		} ],
		"totalResults" : 1,
		"itemsPerPage" : 100,
		"startIndex" : 1,
		"schemas" : [ "urn:ietf:params:scim:api:messages:2.0:ListResponse" ]
	  }`

	OtherGroupResponseBodyWith2MembersAndURLMock = `{
		"Resources" : [ {
		  "id" : "ABC",
		  "meta" : {
			"created" : "2022-02-07T13:44:57Z",
			"lastModified" : "2023-03-03T09:01:28Z",
			"location" : "https://some-location",
			"version" : "3ed780be-888b-4412-9915-661d31e30457",
			"resourceType" : "Group"
		  },
		  "schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:Group", "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" ],
		  "displayName" : "SOME_OTHER_IDP_GROUP_NAME",
		  "members" : [ {
			"value" : "ABC",
			"$ref" : "%s",
			"type" : "User"
		  }, {
			"value" : "BCD",
			"$ref" : "%s",
			"type" : "User"
		  }, {
			"value" : "CDE",
			"$ref" : "%s",
			"type" : "User"
		  }],
		  "urn:sap:cloud:scim:schemas:extension:custom:2.0:Group" : {
			"name" : "SOME_IDP_GROUP_NAME",
			"additionalId" : "123",
			"description" : "SOME_IDP_GROUP_NAME"
		  }
		} ],
		"totalResults" : 1,
		"itemsPerPage" : 100,
		"startIndex" : 1,
		"schemas" : [ "urn:ietf:params:scim:api:messages:2.0:ListResponse" ]
	  }`

	UserResponseBodyMock3 = `{
		"id" : "12345",
		"meta" : {
		  "created" : "2020-09-02T01:19:13Z",
		  "lastModified" : "2022-11-08T00:51:04Z",
		  "location" : "https://some-location",
		  "version" : "123",
		  "resourceType" : "User"
		},
		"schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:User", "urn:ietf:params:scim:schemas:extension:sap:2.0:User" ],
		"userName" : "I9876",
		"name" : {
		  "familyName" : "Mustermann",
		  "givenName" : "Max"
		},
		"displayName" : "John Doe",
		"userType" : "employee",
		"active" : true,
		"emails" : [ {
		  "value" : "max.mustermann@example.com",
		  "primary" : true
		} ],
		"groups" : [ {
		  "value" : "12345",
		  "display" : "SOME_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-location"
		}, {
		  "value" : "23456",
		  "display" : "SOME_OTHER_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-other-location"
		} ],
		"urn:ietf:params:scim:schemas:extension:sap:2.0:User" : {
		  "mock" : "empty"
		}
	  }`

	UserResponseBodyMock4 = `{
		"id" : "12345",
		"meta" : {
		  "created" : "2020-09-02T01:19:13Z",
		  "lastModified" : "2022-11-08T00:51:04Z",
		  "location" : "https://some-location",
		  "version" : "123",
		  "resourceType" : "User"
		},
		"schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:User", "urn:ietf:params:scim:schemas:extension:sap:2.0:User" ],
		"userName" : "I8765",
		"name" : {
		  "familyName" : "Mustermann",
		  "givenName" : "Martina"
		},
		"displayName" : "John Doe",
		"userType" : "employee",
		"active" : true,
		"emails" : [ {
		  "value" : "martina.mustermann@example.com",
		  "primary" : true
		} ],
		"groups" : [ {
		  "value" : "12345",
		  "display" : "SOME_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-location"
		}, {
		  "value" : "23456",
		  "display" : "SOME_OTHER_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-other-location"
		} ],
		"urn:ietf:params:scim:schemas:extension:sap:2.0:User" : {
		  "mock" : "empty"
		}
	  }`

	UserResponseBodyMock5 = `{
		"id" : "12345",
		"meta" : {
		  "created" : "2020-09-02T01:19:13Z",
		  "lastModified" : "2022-11-08T00:51:04Z",
		  "location" : "https://some-location",
		  "version" : "123",
		  "resourceType" : "User"
		},
		"schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:User", "urn:ietf:params:scim:schemas:extension:sap:2.0:User" ],
		"userName" : "I7654",
		"name" : {
		  "familyName" : "Mustermann",
		  "givenName" : "Maxina"
		},
		"displayName" : "John Doe",
		"userType" : "employee",
		"active" : true,
		"emails" : [ {
		  "value" : "maxina.mustermann@example.com",
		  "primary" : true
		} ],
		"groups" : [ {
		  "value" : "12345",
		  "display" : "SOME_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-location"
		}, {
		  "value" : "23456",
		  "display" : "SOME_OTHER_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-other-location"
		} ],
		"urn:ietf:params:scim:schemas:extension:sap:2.0:User" : {
		  "mock" : "empty"
		}
	  }`

	InactiveUserResponseBodyMock = `{
		"id" : "78901",
		"meta" : {
		  "created" : "2020-09-02T01:19:13Z",
		  "lastModified" : "2022-11-08T00:51:04Z",
		  "location" : "https://some-location",
		  "version" : "123",
		  "resourceType" : "User"
		},
		"schemas" : [ "urn:ietf:params:scim:schemas:core:2.0:User", "urn:ietf:params:scim:schemas:extension:sap:2.0:User" ],
		"userName" : "I7654",
		"name" : {
		  "familyName" : "Inactive",
		  "givenName" : "John"
		},
		"displayName" : "John Inactive",
		"userType" : "employee",
		"active" : false,
		"emails" : [ {
		  "value" : "john.inactive@example.com",
		  "primary" : true
		} ],
		"groups" : [ {
		  "value" : "12345",
		  "display" : "SOME_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-location"
		}, {
		  "value" : "23456",
		  "display" : "SOME_OTHER_IDP_GROUP_NAME",
		  "primary" : false,
		  "$ref" : "https://some-other-location"
		} ],
		"urn:ietf:params:scim:schemas:extension:sap:2.0:User" : {
		  "mock" : "empty"
		}
	  }`
)

func returnUserResponseMockServer(bodyMock string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/scim+json")
		_, err := w.Write([]byte(bodyMock))
		if err != nil {
			log.Printf("error creating mock server: %s", err)
		}
	}))
}

func ReturnDefaultGroupResponseMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/scim+json")
		switch r.URL.RawQuery {
		case "filter=displayName+eq+%22SOME_IDP_GROUP_NAME%22":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(fmt.Sprintf(GroupResponseBodyWith2MembersAndURLMock,
				returnUserResponseMockServer(UserResponseBodyMock1).URL,
				returnUserResponseMockServer(UserResponseBodyMock2).URL)))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		case "filter=displayName+eq:+%22SOME_OTHER_IDP_GROUP_NAME%22":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(fmt.Sprintf(OtherGroupResponseBodyWith2MembersAndURLMock,
				returnUserResponseMockServer(UserResponseBodyMock3).URL,
				returnUserResponseMockServer(UserResponseBodyMock4).URL,
				returnUserResponseMockServer(UserResponseBodyMock5).URL)))
			if err != nil {
				log.Printf("error creating mock server: %s", err)
			}
		case "filter=displayName+eq+%22NON_EXISTING_GROUP_NAME%22":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(EmptyGroupResponseBodyMock))
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
	}))
}
