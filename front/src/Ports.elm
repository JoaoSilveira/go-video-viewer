port module Ports exposing (setVolume, toggleFullscreen, togglePlayPause, updateQueryParams)

import Json.Encode as Encode


port togglePlayPausePort : String -> Cmd msg


port setVolumePort : Encode.Value -> Cmd msg


port toggleFullscreenPort : String -> Cmd msg


port updateQueryParamsPort : Encode.Value -> Cmd msg


encodePayload : String -> Float -> Encode.Value
encodePayload id volume =
    Encode.object
        [ ( "id", Encode.string id )
        , ( "volume", Encode.float volume )
        ]


togglePlayPause : String -> Cmd msg
togglePlayPause id =
    togglePlayPausePort id


setVolume : String -> Float -> Cmd msg
setVolume id volume =
    setVolumePort <| encodePayload id volume


toggleFullscreen : String -> Cmd msg
toggleFullscreen id =
    toggleFullscreenPort id


updateQueryParams : List ( String, String ) -> Cmd msg
updateQueryParams params =
    List.map (\( k, v ) -> ( k, Encode.string v )) params
        |> Encode.object
        |> updateQueryParamsPort
