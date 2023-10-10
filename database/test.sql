select contributor_name, pullreq_url, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
		from contributor_record_models as t1
		where contributor_name = ?
		GROUP by pullreq_url, contributor_name 

SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
	select
		contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
	from contributor_record_models as t1
	GROUP by pullreq_url, contributor_name
) GROUP BY contributor_name;

INSERT INTO contributor_models (Name, Current_bounty)
SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
   select
       contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
   from contributor_record_models as t1
   GROUP by pullreq_url, contributor_name
) GROUP BY contributor_name;

-- INSERT INTO contributor_models (Name, Current_bounty, created_at, updated_at)
SELECT cr.*
FROM contributor_record_models cr
         INNER JOIN (
    SELECT Pullreq_url, MAX(created_at) AS latest_created_at
    FROM contributor_record_models
    GROUP BY Pullreq_url
) latest ON cr.Pullreq_url = latest.Pullreq_url AND cr.created_at = latest.latest_created_at;

select contributor_name, pullreq_url, points_allotted, created_at
from contributor_record_models
         where contributor_name like "anuragrao04"
         order by created_at desc;