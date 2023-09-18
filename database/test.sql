select contributor_name, pullreq_url, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
		from contributor_record_models as t1
		where contributor_name = "Smuzzy-waiii"
		GROUP by pullreq_url, contributor_name 

SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
	select
		contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
	from contributor_record_models as t1
	GROUP by pullreq_url, contributor_name
) GROUP BY contributor_name;