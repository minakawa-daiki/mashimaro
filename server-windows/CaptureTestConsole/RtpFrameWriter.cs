using System;
using System.Collections.Generic;
using System.Text;

namespace CaptureTestConsole
{
    class RtpFrameWriter : IFrameWriter
    {
        private readonly IntPtr videoTx;

        private int timestamp;

        private const int timestampStep = 90000 / 60;

        public RtpFrameWriter(string hostname, int port)
        {
            videoTx = NativeMethods.RtpVideoTx_new(-1, VideoFormat.BGRA_8bit);
            if (NativeMethods.RtpVideoTx_addDestination(videoTx, hostname, (uint)port) != 0)
            {
                throw new Exception("failed to videoTx addDestination");
            }
            if (NativeMethods.RtpVideoTx_setSSRC(videoTx, 0) != 0)
            {
                throw new Exception("failed to videoTx setSSRC");
            }
        }

        public void WriteFrame(int width, int height, int rowPitch, IntPtr buffer)
        {
            NativeMethods.RtpVideoTx_beginFrame(videoTx, (uint)timestamp);
            NativeMethods.RtpVideoTx_addFrame(videoTx, width, height, buffer, rowPitch);
            timestamp += timestampStep;
        }
    }
}
