DELETE FROM cluster_nodes WHERE type = "discV5boot" AND node LIKE '%@boot.test.shards.nodes.status.im';

INSERT INTO cluster_nodes(node, type, synthetic_id)
VALUES ("enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im","discV5boot","id");
