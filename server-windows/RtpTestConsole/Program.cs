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
            var writer = new RtpWriter(udpClient, 320, 240, 320);

            byte color = 0x00;
            while (true)
            {
                var frame = Enumerable.Repeat(color, 320 * 240 * 4).ToArray();
                unsafe
                {
                    fixed (byte* p = frame)
                    {
                        try
                        {
                            writer.WriteFrame((IntPtr) p);
                        }
                        catch
                        {
                        }
                    }
                }
                Thread.Sleep(20);
                color++;
            }
        }
    }
}