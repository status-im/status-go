import logging
from time import sleep


# To be used when signals related to RPC requests are not present in the peer ndoe, ex for: requestToJoinCommunity.
def retry_call(func, *args, max_retries=40, retry_interval=0.5, **kwargs):
    for attempt in range(max_retries):
        try:
            response = func(*args, **kwargs)
            if response:
                return response
        except Exception as e:
            logging.error(f"Attempt {attempt + 1}/{max_retries}: Unexpected error: {e}")
        sleep(retry_interval)
    raise Exception(f"Failed to execute {func.__name__} in {max_retries * retry_interval} seconds.")
