module Pages.SavedList exposing (..)

import Browser
import Html exposing (Html, button, div, h1, li, ol, span, text)
import Html.Attributes exposing (class, disabled, href, start, title)
import Html.Events exposing (onClick)
import Http
import Ports exposing (updateQueryParams)
import Util exposing (AsyncResource(..), listSlice, optionalList)
import VideoRepository exposing (Video, listSavedVideos)


type alias VideosPage =
    { videos : List Video
    , page : Int
    , maxPages : Int
    }


type alias SuccModel =
    { videos : List Video
    , page : VideosPage
    }


type alias Model =
    { queryPageNum : Int
    , asyncModel : AsyncResource SuccModel String
    }


type Msg
    = GotVideoList (Result Http.Error (List Video))
    | ToPage Int


init : Maybe Int -> ( Model, Cmd Msg )
init page =
    ( { queryPageNum = Maybe.withDefault 1 page
      , asyncModel = Loading
      }
    , listSavedVideos GotVideoList
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg mdl =
    let
        createPage : List Video -> Int -> VideosPage
        createPage videos page =
            if List.length videos <= 50 then
                { videos = videos, page = 1, maxPages = 1 }

            else
                { videos = listSlice ((page - 1) * 50) 50 videos
                , page = page
                , maxPages = (List.length videos + 49) // 50
                }

        actualPage : Int -> Int -> Int
        actualPage page maxPages =
            Basics.min maxPages <| Basics.max 1 page

        ( newModel, cmd ) =
            case ( mdl.asyncModel, msg ) of
                ( Loading, GotVideoList (Ok videos) ) ->
                    ( Success
                        { videos = videos
                        , page = createPage videos <| actualPage mdl.queryPageNum (List.length videos)
                        }
                    , Cmd.none
                    )

                ( Success model, ToPage page ) ->
                    ( Success
                        { videos = model.videos
                        , page = createPage model.videos <| actualPage page (List.length model.videos)
                        }
                    , updateQueryParams
                        [ ( "page", String.fromInt (actualPage page (List.length model.videos)) ) ]
                    )

                ( _, _ ) ->
                    let
                        invalidStateLog _ =
                            ( mdl.asyncModel, Cmd.none )
                    in
                    invalidStateLog <| Debug.log "UNHANDLED STATE" ( msg, mdl.asyncModel )
    in
    ( { mdl | asyncModel = newModel }, cmd )


view : Model -> Browser.Document Msg
view mdl =
    let
        bodyBase : List (Html Msg) -> Html Msg
        bodyBase content =
            div [ class "container mx-auto p-4" ] <|
                h1 [ class "text-2xl font-bold text-center mb-4" ] [ text "Saved Videos" ]
                    :: content
    in
    case mdl.asyncModel of
        Idle ->
            { title = "Viewed video list"
            , body = [ bodyBase viewLoading ]
            }

        Loading ->
            { title = "Viewed video list"
            , body = [ bodyBase viewLoading ]
            }

        Success model ->
            { title = "Viewed video list"
            , body = [ bodyBase <| viewSuccess model ]
            }

        Failure err ->
            { title = "Viewed video list"
            , body = [ bodyBase <| viewError err ]
            }

        ReFetching (Success model) ->
            { title = "Viewed video list"
            , body = [ bodyBase <| viewSuccess model ]
            }

        ReFetching (Failure err) ->
            { title = "Viewed video list"
            , body = [ bodyBase <| viewError err ]
            }

        ReFetching _ ->
            { title = "Viewed video list"
            , body = [ bodyBase viewLoading ]
            }


viewLoading : List (Html Msg)
viewLoading =
    [ div [ class "text-center" ] [ text "Loading videos..." ] ]


viewError : String -> List (Html Msg)
viewError err =
    [ div [ class "text-red-500" ] [ text err ] ]


viewSuccess : SuccModel -> List (Html Msg)
viewSuccess model =
    [ div [ class "mx-auto max-w-lg shadow-md p-6 rounded-lg" ]
        [ ol
            [ class "space-y-2 list-decimal list-inside"
            , start ((model.page.page - 1) * 50)
            ]
          <|
            List.map videoListItem model.page.videos
        ]
    , paginationControls model
    ]


videoListItem : Video -> Html Msg
videoListItem video =
    let
        tooltip =
            case video.tags of
                [] ->
                    video.filename

                _ ->
                    video.filename ++ " | " ++ String.join ", " video.tags

        linkText =
            "[" ++ String.fromInt video.id ++ "] " ++ Maybe.withDefault video.filename video.nickname
    in
    li []
        [ Html.a
            [ href ("/watch/" ++ String.fromInt video.id)
            , class "text-blue-600 hover:text-blue-800 hover:underline"
            , title tooltip
            ]
            [ text linkText ]
        ]


paginationControls : SuccModel -> Html Msg
paginationControls model =
    let
        currentPage =
            model.page.page

        maxPages =
            model.page.maxPages

        manyItemsPlaceholder =
            span [ class "font-bold py-2" ] [ text "..." ]

        pageButton : Int -> Html Msg
        pageButton page =
            button
                [ class
                    (if page == model.page.page then
                        "bg-blue-700 text-white font-bold py-2 px-4 rounded"

                     else
                        "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
                    )
                , onClick (ToPage page)
                , disabled (page == model.page.page)
                ]
                [ text (String.fromInt page) ]
    in
    div [ class "flex justify-center mt-4 space-x-4" ] <|
        optionalList
            [ ( True, pageButton 1 )
            , ( currentPage > 1 + 3, manyItemsPlaceholder )
            , ( currentPage > 1 + 2, pageButton (currentPage - 2) )
            , ( currentPage > 1 + 1, pageButton (currentPage - 1) )
            , ( currentPage /= 1 && currentPage /= maxPages, pageButton currentPage )
            , ( currentPage + 1 < maxPages, pageButton (currentPage + 1) )
            , ( currentPage + 2 < maxPages, pageButton (currentPage + 2) )
            , ( currentPage + 3 < maxPages, manyItemsPlaceholder )
            , ( True, pageButton maxPages )
            ]
