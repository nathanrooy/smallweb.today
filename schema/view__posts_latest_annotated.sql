create or replace view posts_latest_annotated as
with post_annotations_aggregated as (
	select
		base_url,
		post_url,
		jsonb_object_agg(annotation_type, annotation_values) as annotations
	from (
		select
			base_url,
			post_url,
			annotation_type,
			jsonb_agg(annotation_value) as annotation_values
		from post_annotations
		group by base_url, post_url, annotation_type

	)
	group by base_url, post_url
),

latest_posts as (
    select
    	posts.*,
    	feeds_filtered.sources,
    	feeds_filtered.verified,
    	feeds_filtered.verified_count,
		coalesce(post_annotations_aggregated.annotations ? 'hide', false) as hide,
		coalesce(post_annotations_aggregated.annotations, '{}'::jsonb) as annotations
    from posts
    inner join feeds_filtered using(base_url)
	left join post_annotations_aggregated using(base_url, post_url)
    where
    	-- exclude posts from the future
    	utc_published <= now() at time zone 'utc'

		-- at max, posts from within the last 48 hours
		and utc_published >= now() at time zone 'utc' - interval '48 hours'

    	-- exclude old posts that were just recently discovered
    	and extract(epoch from (utc_discovered - utc_published)) <= (3600 * 8)
)

select *
from latest_posts
order by utc_discovered desc