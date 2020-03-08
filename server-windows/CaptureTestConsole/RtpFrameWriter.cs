using System;
using System.Net.Sockets;
using Rtp;

namespace CaptureTestConsole
{
    public class RtpFrameWriter : IFrameWriter
    {
        private readonly RtpRawVideoWriter _rtpRawVideoWriter;
        
        public RtpFrameWriter(UdpClient udpClient, int width, int height)
        {
            _rtpRawVideoWriter = new RtpRawVideoWriter(udpClient, width, height);
        }
        public void WriteFrame(int width, int height, int rowPitch, IntPtr buffer)
        {
            _rtpRawVideoWriter.WriteFrame(buffer, rowPitch);
        }
    }
}