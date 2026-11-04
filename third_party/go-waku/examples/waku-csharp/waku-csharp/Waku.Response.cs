using System.Runtime.InteropServices;
using System.Text.Json;

namespace Waku
{
    internal static class Response
    {
        internal class JsonResponse<T>
        {
            public string? error { get; set; }
            public T? result { get; set; }
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_utils_free(IntPtr ptr);

        internal static string PtrToStringUtf8(IntPtr ptr, bool free = true) // aPtr is nul-terminated
        {
            if (ptr == IntPtr.Zero)
            {
                waku_utils_free(ptr);
                return "";
            }

            int len = 0;
            while (System.Runtime.InteropServices.Marshal.ReadByte(ptr, len) != 0)
                len++;

            if (len == 0)
            {
                waku_utils_free(ptr);
                return "";
            }

            byte[] array = new byte[len];
            System.Runtime.InteropServices.Marshal.Copy(ptr, array, 0, len);
            string result = System.Text.Encoding.UTF8.GetString(array);

            if (free)
            {
                waku_utils_free(ptr);
            }

            return result;
        }
        
        internal static T HandleResponse<T>(IntPtr ptr, string errNoValue) where T : struct
        {
            string strResponse = PtrToStringUtf8(ptr);
            
            JsonResponse<T?>? response = JsonSerializer.Deserialize<JsonResponse<T?>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (!response.result.HasValue) throw new Exception(errNoValue);

            return response.result.Value;
        }

        internal static void HandleResponse(IntPtr ptr)
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<string>? response = JsonSerializer.Deserialize<JsonResponse<string>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);
        }

        internal static string HandleResponse(IntPtr ptr, string errNoValue)
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<string>? response = JsonSerializer.Deserialize<JsonResponse<string>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (String.IsNullOrEmpty(response.result)) throw new Exception(errNoValue);

            return response.result;
        }

        internal static DecodedPayload HandleDecodedPayloadResponse(IntPtr ptr, string errNoValue)
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<DecodedPayload>? response = JsonSerializer.Deserialize<JsonResponse<DecodedPayload>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (response.result == null) throw new Exception(errNoValue);

            return response.result;
        }

        internal static IList<T> HandleListResponse<T>(IntPtr ptr, string errNoValue)
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<IList<T>>? response = JsonSerializer.Deserialize<JsonResponse<IList<T>>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (response.result == null) throw new Exception(errNoValue);

            return response.result;
        }

        internal static StoreResponse HandleStoreResponse(IntPtr ptr, string errNoValue)
        {
            string strResponse = PtrToStringUtf8(ptr);
            Console.WriteLine(strResponse);
            JsonResponse<StoreResponse>? response = JsonSerializer.Deserialize<JsonResponse<StoreResponse>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (response.result == null) throw new Exception(errNoValue);

            return response.result;
        }
    }
}
