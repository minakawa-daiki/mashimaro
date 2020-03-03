using System;
using System.Collections.Generic;
using System.IO;
using System.Net.Sockets;
using System.Text;

namespace CaptureTestConsole
{
    class TcpFrameWriter : IFrameWriter
    {
        private readonly TcpClient client;

        private readonly BufferedStream stream;
      

        public TcpFrameWriter(string hostname, int port)
        {
            client = new TcpClient();
            client.Connect(hostname, port);
            stream = new BufferedStream(client.GetStream());
        }

        public void WriteFrame(int width, int height, int rowPitch, IntPtr buffer)
        {
            var size = width * 4 * height;
            stream.Write(BitConverter.GetBytes(size).AsSpan());
            unsafe
            {
                for (var row = 0; row < height; row++)
                {
                    var buf = new ReadOnlySpan<byte>(buffer.ToPointer(), width * 4);
                    stream.Write(buf);
                    buffer += rowPitch;
                }
            }
            stream.Flush();
        }
    }
}
