using System;
using System.Diagnostics;
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
            var udpClient = new UdpClient("192.168.10.101", 9999);
            var width = 1282;
            var height = 747;
            var bpp = 4;
            var rowPitch = width * bpp;
            var writer = new RtpRawVideoWriter(udpClient, width, height);
            
            var sw = new Stopwatch();

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
                            sw.Restart();
                            writer.WriteFrame((IntPtr) p, rowPitch);
                            sw.Stop();
                            Console.WriteLine($"frame write: {sw.ElapsedMilliseconds}ms");
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