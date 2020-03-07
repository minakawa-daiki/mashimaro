using System;
using System.Net.Sockets;
using Rtp;

namespace CaptureTestConsole
{
    public class RtpFrameWriter : IFrameWriter
    {
        private readonly RtpWriter rtpWriter;
        
        public RtpFrameWriter(UdpClient udpClient, int width, int height)
        {
            rtpWriter = new RtpWriter(udpClient, width, height);
        }
        public void WriteFrame(int width, int height, int rowPitch, IntPtr buffer)
        {
            rtpWriter.WriteFrame(buffer, rowPitch);
        }
    }
}