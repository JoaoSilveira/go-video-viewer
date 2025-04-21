module Pages.Home exposing (..)

import Browser
import Html exposing (Html, button, div, h1, p, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (class, style)
import Html.Events exposing (onClick)
import Http
import Json.Decode exposing (Decoder, field, int, map4)
import Util exposing (AsyncResource(..), errorToString)


videoStatsUrl : String
videoStatsUrl =
    "http://localhost:8081/api/video/stats"


type alias VideosInfo =
    { unwatched : Int
    , watched : Int
    , liked : Int
    , saved : Int
    }


type alias Model =
    { stats : AsyncResource VideosInfo String
    }


type Msg
    = GotStats (Result Http.Error VideosInfo)
    | FetchStats


videoStatsDecoder : Decoder VideosInfo
videoStatsDecoder =
    let
        statsDecoder : Decoder VideosInfo
        statsDecoder =
            map4 VideosInfo
                (field "unwatched" int)
                (field "watched" int)
                (field "liked" int)
                (field "saved" int)
    in
    field "stats" statsDecoder


init : ( Model, Cmd Msg )
init =
    ( { stats = Loading }
    , Http.get
        { url = videoStatsUrl
        , expect = Http.expectJson GotStats videoStatsDecoder
        }
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case ( msg, model.stats ) of
        ( GotStats (Ok stats), Loading ) ->
            ( { stats = Success stats }
            , Cmd.none
            )

        ( GotStats (Err httpErr), Loading ) ->
            ( { stats = Failure (errorToString httpErr) }
            , Cmd.none
            )

        ( FetchStats, Failure _ ) ->
            ( { stats = Loading }
            , Http.get
                { url = videoStatsUrl
                , expect = Http.expectJson GotStats videoStatsDecoder
                }
            )

        ( _, _ ) ->
            Debug.log "UNHANDLED STATE" ( model, Cmd.none )


view : Model -> Browser.Document Msg
view model =
    let
        actualContent : AsyncResource VideosInfo String -> Html Msg
        actualContent asyncStats =
            case asyncStats of
                Idle ->
                    loadingView

                Loading ->
                    loadingView

                Success stats ->
                    statsView stats

                Failure error ->
                    errorView error

                ReFetching last ->
                    actualContent last
    in
    { title = "Video Stats"
    , body =
        [ div [ class "container mx-auto p-4 flex flex-col items-center" ]
            [ div [ class "w-full max-w-2xl" ] [ actualContent model.stats ] ]
        ]
    }


loadingView : Html Msg
loadingView =
    div [ class "text-center" ]
        [ p [ class "text-gray-500 text-lg" ] [ text "Loading data..." ]
        ]


errorView : String -> Html Msg
errorView error =
    div [ class "text-center space-y-4" ]
        [ p [ class "text-red-500 text-lg" ] [ text ("Error: " ++ error) ]
        , button
            [ class "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
            , onClick FetchStats
            ]
            [ text "Try Again" ]
        ]


statsView : VideosInfo -> Html Msg
statsView stats =
    let
        total =
            stats.unwatched + stats.watched + stats.liked + stats.saved

        percentage stat =
            if total == 0 then
                0

            else
                toFloat stat / toFloat total * 100
    in
    div [ class "w-full" ]
        [ h1
            [ class "text-4xl font-bold text-center mb-8 text-blue-600"
            , style "text-shadow" "2px 2px 4px rgba(0,0,0,0.2)"
            ]
            [ text "Video Stats" ]
        , table [ class "min-w-full border border-gray-300 rounded-lg shadow-md" ]
            [ thead [ class "bg-gray-100 text-gray-600 font-semibold" ]
                [ th [ class "px-4 py-2 text-left" ] [ text "" ]
                , th [ class "px-4 py-2 text-right" ] [ text "*" ]
                , th [ class "px-4 py-2 text-left" ] [ text "%" ]
                ]
            , tbody [ class "bg-white" ]
                [ statRow "Unwatched" stats.unwatched (percentage stats.unwatched)
                , statRow "Watched" stats.watched (percentage stats.watched)
                , statRow "Liked" stats.liked (percentage stats.liked)
                , statRow "Saved" stats.saved (percentage stats.saved)
                , totalRow total
                ]
            ]
        ]


statRow : String -> Int -> Float -> Html Msg
statRow category count percentage =
    tr [ class "hover:bg-gray-50" ]
        [ td [ class "px-4 py-2 text-gray-700" ] [ text category ]
        , td [ class "px-4 py-2 text-gray-900 font-medium text-right" ] [ text (String.fromInt count) ]
        , td [ class "px-4 py-2 text-gray-700" ] [ text (String.fromFloat (roundTo 2 percentage) ++ "%") ]
        ]


totalRow : Int -> Html Msg
totalRow total =
    tr [ class "bg-gray-100 font-bold text-gray-900" ]
        [ td [ class "px-4 py-2" ] [ text "Total" ]
        , td [ class "px-4 py-2 text-right" ] [ text (String.fromInt total) ]
        , td [ class "px-4 py-2" ] [ text "100%" ]
        ]


roundTo : Int -> Float -> Float
roundTo digits num =
    let
        factor : Float
        factor =
            10.0 ^ toFloat digits
    in
    toFloat (round (num * factor)) / factor
