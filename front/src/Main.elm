module Main exposing (..)

import Browser
import Browser.Navigation as Nav
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import Iso8601
import Json.Decode
import Pages.Home
import Pages.NextVideo
import Pages.SavedList
import Pages.WatchVideo
import Routes exposing (routeFromUrl)
import Svg exposing (circle, svg)
import Svg.Attributes exposing (cx, cy, d, fill, r, stroke, strokeDasharray, strokeLinecap, strokeLinejoin, strokeWidth, viewBox)
import Time
import TypedSvg.Attributes
import Url
import Util exposing (AsyncResource(..))
import Routes exposing (areUrlsSamePage)


type PageModel
    = HomePage Pages.Home.Model
    | NextVideoPage Pages.NextVideo.Model
    | WatchVideoPage Pages.WatchVideo.Model
    | SavedListPage Pages.SavedList.Model
    | ErrorPage


type alias Model =
    { key : Nav.Key
    , url : Url.Url
    , lastUpdate : AsyncResource (Maybe Time.Posix) String
    , pageModel : PageModel
    }


type Msg
    = LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url
    | HomeMsg Pages.Home.Msg
    | NextVideoMsg Pages.NextVideo.Msg
    | WatchVideoMsg Pages.WatchVideo.Msg
    | SavedListMsg Pages.SavedList.Msg
    | GotLastUpdate (Result Http.Error (Maybe Time.Posix))
    | RunUpdate


main : Program () Model Msg
main =
    Browser.application
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        , onUrlRequest = LinkClicked
        , onUrlChange = UrlChanged
        }


modelCmdFromRoute : Nav.Key -> (PageModel -> Model) -> Routes.Route -> ( Model, Cmd Msg )
modelCmdFromRoute key makeModel route =
    case route of
        Routes.Home ->
            let
                ( homeModel, homeMsg ) =
                    Pages.Home.init
            in
            ( makeModel (HomePage homeModel), Cmd.map HomeMsg homeMsg )

        Routes.NextVideo ->
            let
                ( queueModel, queueMsg ) =
                    Pages.NextVideo.init key
            in
            ( makeModel (NextVideoPage queueModel), Cmd.map NextVideoMsg queueMsg )

        Routes.WatchVideo id ->
            let
                ( watchModel, watchMsg ) =
                    Pages.WatchVideo.init key id
            in
            ( makeModel (WatchVideoPage watchModel), Cmd.map WatchVideoMsg watchMsg )

        Routes.SavedList page ->
            let
                ( watchModel, watchMsg ) =
                    Pages.SavedList.init page
            in
            ( makeModel (SavedListPage watchModel), Cmd.map SavedListMsg watchMsg )

        Routes.ErrorPage ->
            ( makeModel ErrorPage, Cmd.none )


maybeTimeDecoder : Json.Decode.Decoder (Maybe Time.Posix)
maybeTimeDecoder =
    Json.Decode.field "last_update" <|
        Json.Decode.nullable Iso8601.decoder


fetchLastUpdate : Cmd Msg
fetchLastUpdate =
    Http.get { url = "/api/last-update", expect = Http.expectJson GotLastUpdate maybeTimeDecoder }


sendRunUpdate : Cmd Msg
sendRunUpdate =
    Http.post
        { url = "/api/video/scan"
        , expect = Http.expectJson GotLastUpdate maybeTimeDecoder
        , body = Http.emptyBody
        }


init : () -> Url.Url -> Nav.Key -> ( Model, Cmd Msg )
init _ url key =
    let
        actualUrl =
            case url.path of
                "/src/Main.elm" ->
                    Maybe.withDefault url (Url.fromString "http://localhost:8000/")

                _ ->
                    url

        ( model, pageCmd ) =
            modelCmdFromRoute key (Model key actualUrl Idle) (routeFromUrl actualUrl)
    in
    ( { model | lastUpdate = Loading }
    , Cmd.batch [ pageCmd, fetchLastUpdate ]
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case ( model.pageModel, msg ) of
        ( _, LinkClicked urlRequest ) ->
            case urlRequest of
                Browser.Internal url ->
                    ( model, Nav.pushUrl model.key (Url.toString url) )

                Browser.External href ->
                    ( model, Nav.load href )

        ( _, UrlChanged url ) ->
            let
                ( newModel, cmd ) =
                    modelCmdFromRoute model.key (Model model.key url model.lastUpdate) (routeFromUrl url)
            in
            -- keep current page model
            if areUrlsSamePage model.url url then
                (model, Cmd.none)

            -- update model and init new page
            else
                ( newModel, cmd )

        ( _, GotLastUpdate res ) ->
            case res of
                Ok time ->
                    ( { model
                        | lastUpdate = Success time
                      }
                    , Cmd.none
                    )

                Err err ->
                    ( { model | lastUpdate = Failure <| Util.errorToString err }, Cmd.none )

        ( _, RunUpdate ) ->
            ( { model | lastUpdate = ReFetching model.lastUpdate }
            , sendRunUpdate
            )

        ( HomePage homeModel, HomeMsg homeMsg ) ->
            let
                ( newModel, homeCmd ) =
                    Pages.Home.update homeMsg homeModel
            in
            ( { model | pageModel = HomePage newModel }, Cmd.map HomeMsg homeCmd )

        ( NextVideoPage queueModel, NextVideoMsg queueMsg ) ->
            let
                ( newModel, queueCmd ) =
                    Pages.NextVideo.update queueMsg queueModel
            in
            ( { model | pageModel = NextVideoPage newModel }, Cmd.map NextVideoMsg queueCmd )

        ( WatchVideoPage watchModel, WatchVideoMsg watchMsg ) ->
            let
                ( newModel, queueCmd ) =
                    Pages.WatchVideo.update watchMsg watchModel
            in
            ( { model | pageModel = WatchVideoPage newModel }, Cmd.map WatchVideoMsg queueCmd )

        ( SavedListPage watchModel, SavedListMsg watchMsg ) ->
            let
                ( newModel, queueCmd ) =
                    Pages.SavedList.update watchMsg watchModel
            in
            ( { model | pageModel = SavedListPage newModel }, Cmd.map SavedListMsg queueCmd )

        ( ErrorPage, _ ) ->
            ( model, Cmd.none )

        ( _, _ ) ->
            let
                logInvalidState _ =
                    ( model, Cmd.none )
            in
            logInvalidState <| Debug.log "INVALID STATE" ( msg, model )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none


view : Model -> Browser.Document Msg
view model =
    let
        mapPage : (a -> Msg) -> List (Html a) -> List (Html Msg)
        mapPage msg body =
            navbar model.url model.lastUpdate :: List.map (Html.map msg) body
    in
    case model.pageModel of
        HomePage homeModel ->
            let
                { title, body } =
                    Pages.Home.view homeModel
            in
            { title = title
            , body = mapPage HomeMsg body
            }

        NextVideoPage queueModel ->
            let
                { title, body } =
                    Pages.NextVideo.view queueModel
            in
            { title = title
            , body = mapPage NextVideoMsg body
            }

        WatchVideoPage watchModel ->
            let
                { title, body } =
                    Pages.WatchVideo.view watchModel
            in
            { title = title
            , body = mapPage WatchVideoMsg body
            }

        SavedListPage listModel ->
            let
                { title, body } =
                    Pages.SavedList.view listModel
            in
            { title = title
            , body = mapPage SavedListMsg body
            }

        ErrorPage ->
            { title = "Error!"
            , body = [ navbar model.url model.lastUpdate ]
            }


navbar : Url.Url -> AsyncResource (Maybe Time.Posix) String -> Html Msg
navbar url lastUpdate =
    let
        timeDescription : Maybe Time.Posix -> Html msg
        timeDescription time =
            span [ class "text-gray-500 text-sm" ]
                [ text <|
                    case time of
                        Just posix ->
                            Iso8601.fromTime posix

                        Nothing ->
                            "Never updated"
                ]

        refreshButton : Bool -> Html Msg
        refreshButton running =
            button
                [ class "text-gray-700 hover:text-blue-600 cursor-pointer"
                , onClick RunUpdate
                , disabled running
                ]
                [ refreshIcon ]

        lastUpdatedPart : AsyncResource (Maybe Time.Posix) String -> Bool -> List (Html Msg)
        lastUpdatedPart last running =
            case last of
                Idle ->
                    [ circularLoadingIndicator ]

                Loading ->
                    [ circularLoadingIndicator ]

                Success time ->
                    [ timeDescription time
                    , refreshButton running
                    ]

                Failure err ->
                    [ span [ class "text-gray-500" ] [ text err ]
                    , refreshButton running
                    ]

                ReFetching asyncState ->
                    circularLoadingIndicator :: lastUpdatedPart asyncState True

        pageLink : String -> String -> Bool -> Html msg
        pageLink pageHref txt currentPage =
            li [] <|
                [ a
                    [ href pageHref
                    , classList
                        [ ( "hover:text-blue-600", True )
                        , ( "text-blue-600", currentPage )
                        ]
                    ]
                    [ text txt ]
                ]
    in
    nav [ class "bg-white shadow-md py-4" ]
        [ div [ class "container mx-auto flex justify-between items-center" ]
            [ ul [ class "flex space-x-6 text-gray-700" ]
                [ pageLink
                    (Routes.routeHref Routes.Home)
                    "Stats"
                    (Routes.isRouteActive url Routes.Home)
                , pageLink
                    (Routes.routeHref Routes.NextVideo)
                    "Queue"
                    (Routes.isRouteActive url Routes.NextVideo)
                , pageLink
                    (Routes.routeHref (Routes.SavedList Nothing))
                    "Saved"
                    (Routes.isRouteActive url (Routes.SavedList Nothing))
                ]
            , div [ class "flex items-center gap-2" ] <|
                lastUpdatedPart lastUpdate False
            ]
        ]


circularLoadingIndicator : Html msg
circularLoadingIndicator =
    svg
        [ TypedSvg.Attributes.class [ "animate-spin", "h-5", "w-5", "text-blue-500" ]
        , viewBox "0 0 100 100"
        ]
        [ circle
            [ cx "50"
            , cy "50"
            , r "45"
            , stroke "currentColor"
            , strokeWidth "4"
            , strokeDasharray "25,75"
            , fill "none"
            , attribute "style" "stroke-linecap: round"
            ]
            []
        ]


refreshIcon : Html msg
refreshIcon =
    svg
        [ viewBox "0 0 24 24"
        , stroke "currentColor"
        , strokeWidth "0.75"
        , fill "none"
        , TypedSvg.Attributes.class [ "h-5", "w-5" ]
        ]
        [ Svg.g
            [ strokeLinecap "round"
            , strokeLinejoin "round"
            ]
            [ Svg.path
                [ d "M2.5 12C2.5 12.2761 2.72386 12.5 3 12.5C3.27614 12.5 3.5 12.2761 3.5 12H2.5ZM3.5 12C3.5 7.30558 7.30558 3.5 12 3.5V2.5C6.75329 2.5 2.5 6.75329 2.5 12H3.5ZM12 3.5C15.3367 3.5 18.2252 5.4225 19.6167 8.22252L20.5122 7.77748C18.9583 4.65062 15.7308 2.5 12 2.5V3.5Z"
                , fill "currentColor"
                ]
                []
            , Svg.path
                [ d "M20.4716 2.42157V8.07843H14.8147"
                , fill "currentColor"
                ]
                []
            , Svg.path
                [ d "M21.5 12C21.5 11.7239 21.2761 11.5 21 11.5C20.7239 11.5 20.5 11.7239 20.5 12L21.5 12ZM20.5 12C20.5 16.6944 16.6944 20.5 12 20.5L12 21.5C17.2467 21.5 21.5 17.2467 21.5 12L20.5 12ZM12 20.5C8.66333 20.5 5.77477 18.5775 4.38328 15.7775L3.48776 16.2225C5.04168 19.3494 8.26923 21.5 12 21.5L12 20.5Z"
                , fill "currentColor"
                ]
                []
            , Svg.path
                [ d "M3.52844 21.5784L3.52844 15.9216L9.18529 15.9216"
                , fill "currentColor"
                ]
                []
            ]
        ]
