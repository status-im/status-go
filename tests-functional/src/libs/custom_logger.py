import logging

max_log_line_length = 10000


def log_length_filter(max_length):
    class logLengthFilter(logging.Filter):
        def filter(self, record):
            if len(record.getMessage()) > max_length:
                logging.getLogger(record.name).log(
                    record.levelno, f"Log line was discarded because it's longer than max_log_line_length={max_log_line_length}"
                )
                return False
            return True

    return logLengthFilter()


def get_custom_logger(name):
    logging.getLogger("urllib3").setLevel(logging.WARNING)
    logging.getLogger("docker").setLevel(logging.WARNING)
    logger = logging.getLogger(name)
    logger.addFilter(log_length_filter(max_log_line_length))
    return logger
