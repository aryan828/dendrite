// Copyright 2020 New Vector Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package routing

import (
	"net/http"

	"github.com/matrix-org/dendrite/clientapi/jsonerror"
	"github.com/matrix-org/dendrite/internal/config"
	"github.com/matrix-org/dendrite/roomserver/api"
	"github.com/matrix-org/gomatrixserverlib"
	"github.com/matrix-org/util"
)

// Peek implements the SS /peek API
func Peek(
	httpReq *http.Request,
	request *gomatrixserverlib.FederationRequest,
	cfg *config.FederationAPI,
	rsAPI api.RoomserverInternalAPI,
	roomID, peekID string,
	remoteVersions []gomatrixserverlib.RoomVersion,
) util.JSONResponse {
	// TODO: check if we're just refreshing an existing peek. Somehow.

	verReq := api.QueryRoomVersionForRoomRequest{RoomID: roomID}
	verRes := api.QueryRoomVersionForRoomResponse{}
	if err := rsAPI.QueryRoomVersionForRoom(httpReq.Context(), &verReq, &verRes); err != nil {
		return util.JSONResponse{
			Code: http.StatusInternalServerError,
			JSON: jsonerror.InternalServerError(),
		}
	}

	// Check that the room that the peeking server is trying to peek is actually
	// one of the room versions that they listed in their supported ?ver= in
	// the peek URL.
	remoteSupportsVersion := false
	for _, v := range remoteVersions {
		if v == verRes.RoomVersion {
			remoteSupportsVersion = true
			break
		}
	}
	// If it isn't, stop trying to peek the room.
	if !remoteSupportsVersion {
		return util.JSONResponse{
			Code: http.StatusBadRequest,
			JSON: jsonerror.IncompatibleRoomVersion(verRes.RoomVersion),
		}
	}

	// TODO: Check history visibility

	var response api.PerformHandleRemotePeekResponse
	err := rsAPI.PerformHandleRemotePeek(
		httpReq.Context(),
		&api.PerformHandleRemotePeekRequest{
			RoomID:       roomID,
			PeekID:		  peekID,
			ServerName:	  request.Origin(),
		},
		&response,
	)
	if err != nil {
		resErr := util.ErrorResponse(err)
		return resErr
	}

	if !response.RoomExists {
		return util.JSONResponse{Code: http.StatusNotFound, JSON: nil}
	}

	return util.JSONResponse{
		Code: http.StatusOK,
		JSON: gomatrixserverlib.RespPeek{
			StateEvents: gomatrixserverlib.UnwrapEventHeaders(response.StateEvents),
			AuthEvents:  gomatrixserverlib.UnwrapEventHeaders(response.AuthChainEvents),
			RoomVersion: response.RoomVersion,
			RenewalInterval: 60 * 60 * 1000 * 1000, // one hour
		},
	}
}

