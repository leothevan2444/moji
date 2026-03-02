
--
-- Name: derived_actor; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_actor (
    id integer NOT NULL,
    name_kanji character varying,
    name_kana character varying
);


--
-- Name: derived_actress; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_actress (
    id integer NOT NULL,
    name_romaji character varying,
    image_url character varying,
    name_kanji character varying,
    name_kana character varying
);


--
-- Name: derived_author; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_author (
    id integer NOT NULL,
    name_kanji character varying,
    name_kana character varying
);


--
-- Name: derived_category; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_category (
    id integer NOT NULL,
    name_en character varying,
    name_ja character varying
);


--
-- Name: derived_director; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_director (
    id integer NOT NULL,
    name_kanji character varying,
    name_kana character varying,
    name_romaji character varying
);


--
-- Name: derived_label; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_label (
    id integer NOT NULL,
    name_en character varying,
    name_ja character varying
);


--
-- Name: derived_maker; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_maker (
    id integer NOT NULL,
    name_en character varying,
    name_ja character varying
);


--
-- Name: derived_series; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_series (
    id integer NOT NULL,
    name_en character varying,
    name_ja character varying
);


--
-- Name: derived_site; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_site (
    id integer NOT NULL,
    name character varying NOT NULL
);


--
-- Name: derived_video; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_video (
    content_id character varying NOT NULL,
    dvd_id character varying,
    title_en character varying,
    title_ja character varying,
    comment_en character varying,
    comment_ja character varying,
    runtime_mins integer,
    release_date date,
    sample_url character varying,
    maker_id integer,
    label_id integer,
    series_id integer,
    jacket_full_url character varying,
    jacket_thumb_url character varying,
    gallery_full_first character varying,
    gallery_full_last character varying,
    gallery_thumb_first character varying,
    gallery_thumb_last character varying,
    site_id integer NOT NULL,
    service_code character varying NOT NULL
);


--
-- Name: derived_video_actor; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_video_actor (
    content_id character varying NOT NULL,
    actor_id integer NOT NULL,
    ordinality integer,
    release_date date
);


--
-- Name: derived_video_actress; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_video_actress (
    content_id character varying NOT NULL,
    actress_id integer NOT NULL,
    ordinality integer,
    release_date date
);


--
-- Name: derived_video_author; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_video_author (
    content_id character varying NOT NULL,
    author_id integer NOT NULL
);


--
-- Name: derived_video_category; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_video_category (
    content_id character varying NOT NULL,
    category_id integer NOT NULL,
    release_date date
);


--
-- Name: derived_video_director; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.derived_video_director (
    content_id character varying NOT NULL,
    director_id integer NOT NULL
);


--
-- Name: machine_translation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.machine_translation (
    id integer NOT NULL,
    source_ja character varying NOT NULL,
    target_en character varying NOT NULL,
    "timestamp" timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: machine_translation_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.machine_translation ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.machine_translation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: source_dmm_histrion; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.source_dmm_histrion (
    id integer NOT NULL,
    name_kanji character varying,
    created timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    name_kanji_only character varying,
    name_kana character varying
);


--
-- Name: source_dmm_trailer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.source_dmm_trailer (
    content_id character varying NOT NULL,
    url character varying,
    "timestamp" timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: source_dmm_video_histrion; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.source_dmm_video_histrion (
    histrion_id integer NOT NULL,
    "timestamp" timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    content_id character varying NOT NULL
);
