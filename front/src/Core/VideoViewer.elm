module Core.VideoViewer exposing (..)

import Browser.Navigation as Navigation
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (..)
import Http
import Iso8601
import Json.Decode as Decode
import Ports
import Routes
import Util exposing (AsyncResource(..), errorToString, optionalList)
import VideoRepository exposing (Video, VideoInfo, VideoStatus(..), VideoUpdatePayload, getVideoById, updateVideo)


type alias FormState =
    { nickname : String
    , status : VideoStatus
    , tags : String
    , volume : Float
    }


type alias Model =
    { video : AsyncResource VideoInfo String
    , formState : FormState
    , videoUpdate : AsyncResource () String
    , key : Navigation.Key
    , changesUrl : Bool
    }


type Msg
    = GotVideoInfo (Result Http.Error VideoInfo)
    | UpdatedVideo (Result Http.Error ())
    | NicknameChanged String
    | StatusChanged VideoStatus
    | TagsChanged String
    | SubmitForm
    | TogglePlayPause
    | VolumeChanged Float
    | ToggleFullscreen


init : Navigation.Key -> Bool -> ((Result Http.Error VideoInfo -> Msg) -> Cmd Msg) -> ( Model, Cmd Msg )
init key changesUrl getter =
    ( { video = Loading
      , formState =
            { nickname = ""
            , status = Watched
            , tags = ""
            , volume = 1.0
            }
      , videoUpdate = Idle
      , key = key
      , changesUrl = changesUrl
      }
    , getter GotVideoInfo
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    let
        formState =
            model.formState

        formStatus status =
            case status of
                Unwatched ->
                    Watched

                _ ->
                    status

        newFormState video =
            { nickname = Maybe.withDefault "" video.nickname
            , status = formStatus video.status
            , tags = String.join "," video.tags
            , volume = formState.volume
            }
    in
    case ( msg, model.video ) of
        ( GotVideoInfo (Ok videoInfo), Loading ) ->
            ( { model
                | video = Success videoInfo
                , formState = newFormState videoInfo.video
              }
            , Cmd.none
            )

        ( GotVideoInfo (Ok videoInfo), ReFetching _ ) ->
            ( { model
                | video = Success videoInfo
                , formState = newFormState videoInfo.video
              }
            , Cmd.none
            )

        ( UpdatedVideo (Ok _), Success _ ) ->
            ( { model
                | videoUpdate = Success ()
              }
            , Cmd.none
            )

        ( UpdatedVideo (Ok _), ReFetching (Success _) ) ->
            ( { model
                | videoUpdate = Success ()
              }
            , Cmd.none
            )

        ( GotVideoInfo (Err err), Loading ) ->
            ( { model | video = Failure (errorToString err) }, Cmd.none )

        ( GotVideoInfo (Err err), ReFetching _ ) ->
            ( { model | video = Failure (errorToString err) }, Cmd.none )

        ( NicknameChanged nickname, Success _ ) ->
            ( { model | formState = { formState | nickname = nickname } }, Cmd.none )

        ( StatusChanged status, Success _ ) ->
            ( { model | formState = { formState | status = status } }, Cmd.none )

        ( TagsChanged tags, Success _ ) ->
            ( { model | formState = { formState | tags = tags } }, Cmd.none )

        ( SubmitForm, Success video ) ->
            let
                payload : VideoUpdatePayload
                payload =
                    { nickname = model.formState.nickname
                    , status = model.formState.status
                    , tags = String.split "," model.formState.tags
                    }

                updateVideoCommand : Cmd Msg
                updateVideoCommand =
                    updateVideo video.video.id payload UpdatedVideo
            in
            case video.next of
                Just next ->
                    ( { model
                        | video = ReFetching (Success (VideoInfo next Nothing))
                        , formState = newFormState next
                        , videoUpdate = Loading
                      }
                    , Cmd.batch <|
                        optionalList
                            [ ( True, updateVideoCommand )
                            , ( True, getVideoById next.id GotVideoInfo )
                            , ( model.changesUrl, Navigation.replaceUrl model.key (Routes.routeHref (Routes.WatchVideo next.id)) )
                            ]
                    )

                Nothing ->
                    ( { model | videoUpdate = Loading }
                    , updateVideoCommand
                    )

        ( TogglePlayPause, Success _ ) ->
            ( model, Ports.togglePlayPause "video" )

        ( VolumeChanged volume, Success _ ) ->
            ( { model | formState = { formState | volume = volume } }, Ports.setVolume "video" volume )

        ( ToggleFullscreen, Success _ ) ->
            ( model, Ports.toggleFullscreen "video" )

        ( _, _ ) ->
            let
                invalidStateLog _ =
                    ( model, Cmd.none )
            in
            invalidStateLog <| Debug.log "UNHANDLED STATE" ( msg, model.video )


view : Model -> Html Msg
view model =
    let
        isUpdating =
            case model.videoUpdate of
                Loading ->
                    True

                _ ->
                    False

        viewFromModel : AsyncResource VideoInfo String -> Bool -> Html Msg
        viewFromModel video fetching =
            case video of
                Idle ->
                    loadingView

                Loading ->
                    loadingView

                Success videoInfo ->
                    successView model.formState videoInfo (fetching || isUpdating)

                Failure err ->
                    errView err fetching

                ReFetching lastState ->
                    viewFromModel lastState True
    in
    viewFromModel model.video False


loadingView : Html Msg
loadingView =
    div [ class "container mx-auto p-4" ]
        [ div [ class "text-center" ] [ text "Loading video..." ] ]


errView : String -> Bool -> Html Msg
errView err loading =
    div [ class "container mx-auto p-4" ] <|
        if loading then
            [ div [ class "text-center" ] [ text err ]
            , div [ class "text-center" ] [ text "Trying again..." ]
            ]

        else
            [ div [ class "text-center" ] [ text err ] ]


successView : FormState -> VideoInfo -> Bool -> Html Msg
successView state videoInfo cantUpdate =
    div [ class "container mx-auto p-4" ]
        [ videoView state videoInfo.video
        , formView state videoInfo cantUpdate
        ]


videoView : FormState -> Video -> Html Msg
videoView formState video =
    let
        volumeStep =
            0.05

        adjstVolume delta =
            if delta > 0 then
                formState.volume - volumeStep

            else
                formState.volume + volumeStep

        volumeEventDecoder =
            Decode.field "deltaY" (Decode.map adjstVolume Decode.float)
                |> Decode.map (\v -> Basics.max 0 <| Basics.min 1 v)
                |> Decode.map VolumeChanged
                |> Decode.map (\msg -> { message = msg, stopPropagation = True, preventDefault = True })

        fullscreenEventDecoder =
            Decode.field "key" Decode.string
                |> Decode.andThen
                    (\k ->
                        if k == "F" || k == "f" then
                            Decode.succeed ToggleFullscreen

                        else
                            Decode.fail "no message"
                    )
                |> Decode.map (\msg -> { message = msg, stopPropagation = True, preventDefault = True })
    in
    div []
        [ div [ class "flex justify-center my-8" ]
            [ Html.video
                [ id "video"
                , class "rounded-lg shadow-xl"
                , style "width" "70%"
                , style "max-width" "800px"
                , style "height" "auto"
                , style "max-height" "70vh"
                , src ("/api/video/" ++ String.fromInt video.id ++ "/serve")
                , controls True
                , custom "wheel" volumeEventDecoder
                , custom "keydown" fullscreenEventDecoder
                ]
                []
            ]
        ]


formView : FormState -> VideoInfo -> Bool -> Html Msg
formView state videoInfo cantUpdate =
    let
        video =
            videoInfo.video
    in
    Html.form
        [ class "w-full max-w-lg mx-auto bg-white p-6 rounded-lg shadow-md"
        , custom "submit"
            (Decode.succeed
                { message = SubmitForm
                , preventDefault = True
                , stopPropagation = True
                }
            )
        ]
        [ div [ class "text-center mb-4" ]
            [ h1 [ class "text-lg font-semibold text-gray-800" ] [ text ("ID: " ++ String.fromInt video.id) ]
            , p [ class "text-sm text-gray-600" ] [ text ("File: " ++ video.filename) ]
            , p [ class "text-xs text-gray-500" ] [ text ("Created: " ++ Iso8601.fromTime video.created_at) ]
            ]
        , div [ class "mb-4" ]
            [ input
                [ type_ "text"
                , id "nickname"
                , class "shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
                , value state.nickname
                , placeholder "Nickname"
                , onInput NicknameChanged
                ]
                []
            ]
        , fieldset [ class "mb-4" ]
            [ legend [ class "text-sm font-bold text-gray-700" ] [ text "Status" ]
            , div [ class "mt-2 flex space-x-4" ] <|
                List.map
                    (\( status, labelText ) ->
                        div [ class "flex items-center" ]
                            [ input
                                [ type_ "radio"
                                , id (String.toLower labelText)
                                , class "h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 focus:ring-0"
                                , value (String.toLower labelText)
                                , checked (state.status == status)
                                , on "change" (Decode.succeed (StatusChanged status))
                                ]
                                []
                            , label
                                [ for (String.toLower labelText)
                                , class "ml-2 text-sm font-medium text-gray-900"
                                ]
                                [ text labelText ]
                            ]
                    )
                    [ ( Watched, "Watched" )
                    , ( Liked, "Liked" )
                    , ( Saved, "Saved" )
                    ]
            ]
        , div [ class "mb-6" ]
            [ input
                [ type_ "text"
                , id "tags"
                , class "shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
                , value state.tags
                , placeholder "Tags (comma-separated)"
                , onInput TagsChanged
                ]
                []
            ]
        , button
            [ class "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline w-full"
            , type_ "submit"
            , disabled cantUpdate
            ]
            [ text "Save & Next" ]
        ]
