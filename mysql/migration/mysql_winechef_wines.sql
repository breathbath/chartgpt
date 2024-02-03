# ************************************************************
# Sequel Ace SQL dump
# Version 20056
#
# https://sequel-ace.com/
# https://github.com/Sequel-Ace/Sequel-Ace
#
# Host: localhost (MySQL 8.2.0)
# Database: winechefV2
# Generation Time: 2024-02-03 17:00:27 +0000
# ************************************************************


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
SET NAMES utf8mb4;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE='NO_AUTO_VALUE_ON_ZERO', SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


# Dump of table usage_stats
# ------------------------------------------------------------

DROP TABLE IF EXISTS `usage_stats`;

CREATE TABLE `usage_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `last_name` varchar(255) DEFAULT NULL,
  `user_id` varchar(255) DEFAULT NULL,
  `first_name` varchar(255) DEFAULT NULL,
  `session_start` datetime(3) DEFAULT NULL,
  `session_end` datetime(3) DEFAULT NULL,
  `input_prompt_tokens` bigint DEFAULT NULL,
  `input_completion_tokens` bigint DEFAULT NULL,
  `gpt_model` varchar(255) DEFAULT NULL,
  `tracking_id` varchar(255) DEFAULT NULL,
  `error` longtext,
  `is_voice_input` tinyint(1) DEFAULT NULL,
  `voice_to_text_model` varchar(255) DEFAULT NULL,
  `input` longtext,
  `gen_prompt_tokens` bigint DEFAULT NULL,
  `gen_completion_tokens` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_usage_stats_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

LOCK TABLES `usage_stats` WRITE;
/*!40000 ALTER TABLE `usage_stats` DISABLE KEYS */;

INSERT INTO `usage_stats` (`id`, `created_at`, `updated_at`, `deleted_at`, `last_name`, `user_id`, `first_name`, `session_start`, `session_end`, `input_prompt_tokens`, `input_completion_tokens`, `gpt_model`, `tracking_id`, `error`, `is_voice_input`, `voice_to_text_model`, `input`, `gen_prompt_tokens`, `gen_completion_tokens`)
VALUES
	(1,'2024-02-03 12:30:18.293','2024-02-03 12:30:18.293',NULL,'','LALA','Anjey','2024-02-03 12:30:13.367','2024-02-03 12:30:18.292',247,32,'gpt-3.5-turbo-16k-0613','692b808e-b1f3-4ae1-b7b1-c40adef2497c','',0,'','сладкое',109,217),
	(2,'2024-02-03 12:31:14.661','2024-02-03 12:31:14.661',NULL,'','Andzeibodzey','Anjey','2024-02-03 12:31:10.356','2024-02-03 12:31:14.649',483,34,'gpt-3.5-turbo-16k-0613','d02f005e-0277-4851-a8ca-aee0577be691','',1,'whisper-1','А как насчет полусухого?',109,171),
	(3,'2024-02-03 12:40:12.493','2024-02-03 12:40:12.493',NULL,'','Andzeibodzey','Anjey','2024-02-03 12:40:07.816','2024-02-03 12:40:12.493',674,35,'gpt-3.5-turbo-16k-0613','f73c1b49-ca10-4c0d-bd0d-4cb223ab2005','',0,'','красное полусухое вино',109,232),
	(4,'2024-02-03 13:01:27.274','2024-02-03 13:01:27.274',NULL,'','Andzeibodzey','Anjey','2024-02-03 13:01:23.880','2024-02-03 13:01:27.266',262,120,'gpt-3.5-turbo-16k-0613','8fbba568-55d5-4823-901f-399311228634','',1,'whisper-1','Это сухо-белая вина, а что ещё?',0,0),
	(5,'2024-02-03 13:01:50.384','2024-02-03 13:01:50.384',NULL,'','LALA','Anjey','2024-02-03 13:01:42.770','2024-02-03 13:01:50.381',401,32,'gpt-3.5-turbo-16k-0613','e35d7567-cf7d-423b-829a-10c49a8b46d5','',1,'whisper-1','Ну давай сухое белое',524,423),
	(6,'2024-02-03 13:04:03.396','2024-02-03 13:04:03.396',NULL,'','MAMA','Anjey','2024-02-03 13:03:58.635','2024-02-03 13:04:03.396',837,32,'gpt-3.5-turbo-16k-0613','2cd4c4f1-7681-4d8d-a2a8-066b2e690833','',1,'whisper-1','А если другое?',543,254),
	(7,'2024-02-03 13:05:04.166','2024-02-03 13:05:04.166',NULL,'','Andzeibodzey','Anjey','2024-02-03 13:04:59.942','2024-02-03 13:05:04.162',1103,32,'gpt-3.5-turbo-16k-0613','971fd7e7-f80f-4bcd-8d53-645e727c86b0','',0,'','или другое',409,246),
	(8,'2024-02-03 13:07:16.312','2024-02-03 13:07:16.312',NULL,'','Andzeibodzey','Anjey','2024-02-03 13:07:09.184','2024-02-03 13:07:16.291',1361,32,'gpt-3.5-turbo-16k-0613','a100add6-7afe-43a0-9676-d2778e45ac36','',0,'','или другое',632,361),
	(9,'2024-02-03 14:31:06.694','2024-02-03 14:31:06.694',NULL,'','Andzeibodzey','Anjey','2024-02-03 14:31:05.169','2024-02-03 14:31:06.688',252,61,'gpt-3.5-turbo-16k-0613','4eeddf58-b7af-42d8-b9b5-34afe4d7996d','',0,'','мне бы красного вина',0,0),
	(10,'2024-02-03 14:31:13.887','2024-02-03 14:31:13.887',NULL,'','Andzeibodzey','Anjey','2024-02-03 14:31:10.767','2024-02-03 14:31:13.887',324,33,'gpt-3.5-turbo-16k-0613','fd7aaa23-87c1-43d0-87bd-9a9d42420452','',0,'','сладкое',464,176);

/*!40000 ALTER TABLE `usage_stats` ENABLE KEYS */;
UNLOCK TABLES;



/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
