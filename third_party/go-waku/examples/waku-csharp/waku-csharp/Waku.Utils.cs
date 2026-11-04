using System.Runtime.InteropServices;

namespace Waku
{
    public static class Utils
    {
        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_default_pubsub_topic();

        /// <summary>
        /// Get default pubsub topic
        /// </summary>
        /// <returns>Default pubsub topic used for exchanging waku messages defined in RFC 10</returns>
        public static string DefaultPubsubTopic()
        {
            IntPtr ptr = waku_default_pubsub_topic();
            return Response.PtrToStringUtf8(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr waku_content_topic(string applicationName, uint applicationVersion, string contentTopicName, string encoding);

        /// <summary>
        /// Create a content topic string
        /// </summary>
        /// <param name="applicationName"></param>
        /// <param name="applicationVersion"></param>
        /// <param name="contentTopicName"></param>
        /// <param name="encoding"></param>
        /// <returns>Content topic string according to RFC 23</returns>
        public static string ContentTopic(string applicationName, uint applicationVersion, string contentTopicName, string encoding)
        {
            IntPtr ptr = waku_content_topic(applicationName, applicationVersion, contentTopicName, encoding);
            return Response.PtrToStringUtf8(ptr);
        }

    }
}
