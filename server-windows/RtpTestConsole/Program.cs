using System;
using System.Linq;
using System.Net.Sockets;
using System.Threading;
using Rtp;

namespace RtpTestConsole
{
    class Program
    {
        static void Main(string[] args)
        {
            var udpClient = new UdpClient("127.0.0.1", 9999);
            var width = 320;
            var height = 240;
            var bpp = 4;
            var rowPitch = width * bpp;
            var writer = new RtpWriter(udpClient, width, height, rowPitch);

            byte color = 0xff;
            for (var i = 0; i < 200000; i++)
            {
                var frame = Enumerable.Repeat(color, width * height * bpp).ToArray();
                unsafe
                {
                    fixed (byte* p = frame)
                    {
                        try
                        {
                            writer.WriteFrame((IntPtr) p);
                        }
                        catch (SocketException ex)
                        {
                            Console.WriteLine($"socket exception: {ex.Message}");
                        }
                    }
                }
                Thread.Sleep(20);
                color--;
            }
        }
    }
}