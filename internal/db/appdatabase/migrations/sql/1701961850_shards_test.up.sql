UPDATE settings SET fleet = "shards.test" WHERE fleet = "" OR fleet IS NULL;

UPDATE node_config SET rendezvous = false;

UPDATE wakuv2_config SET use_shard_default_topic = true;

UPDATE cluster_config SET fleet = "shards.test";
DELETE FROM cluster_nodes;

UPDATE cluster_config SET cluster_id = 16;

INSERT INTO cluster_nodes(node, type, synthetic_id)
VALUES ("enr:-Ni4QAG-O7ryJQg1P-CDwE7nBoSx-pScZsRRq6tvBF0tRsCGFtbs2ag1bqsv7GpTD_2rTvwIT7PsOVNG_ytFZdfwT3cBgmlkgnY0gmlwhKdjEy-KbXVsdGl","discV5boot","id");

INSERT INTO cluster_nodes(node, type, synthetic_id)
VALUES ("enr:-OK4QFH-vPVmsKjlEd3jjS8heib42DO5ZGNVUYM-lbJkPL2QSP0Ye8VZV-WycXk8jVjv9LcQpuwlaBJ3xN1ttPMy07wBgmlkgnY0gmlwhAjaF0yKbXVsdGl","discV5boot","id");

INSERT INTO cluster_nodes(node, type, synthetic_id)
VALUES ("enr:-OK4QFWlB2csVi4NhszuVmzOWd1q1Moy1DFTmq1Bt4_AWKh7U-eCRHTj3m9TOma53DLXN318cS7LapchI01ZxnEwLXEBgmlkgnY0gmlwhCKHDVeKbXVsdGl","discV5boot","id");

INSERT INTO cluster_nodes(node, type, synthetic_id)
VALUES ("enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im","discV5boot","id");

INSERT INTO cluster_nodes(node, type, synthetic_id)
VALUES ("enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im","waku","id");

