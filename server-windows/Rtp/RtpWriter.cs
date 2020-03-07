using System;
using System.IO;
using System.Net.Sockets;

namespace Rtp
{
    public class RtpWriter
    {
        private readonly UdpClient udpClient;
        private readonly MemoryStream sendBuffer = new MemoryStream(Mtu);
        
        private const int RtpHeaderSize = 12;
        private const int Rfc4175HeaderSize = 2;
        private const int Rfc4175SegmentHeaderSize = 6;
            
        private const ushort Mtu = 9000;
        private const int Bpp = 4; // RGBA: 4 bytes / pixel
        
        private readonly int frameWidth;
        private readonly int frameHeight;
        private readonly int frameRowPitch;
        
        private byte[] rtpHeader;
        private short seqNo;
        private int timestamp;
        private int ssrc;

        private byte[] payloadHeader;
        private const short extSeqNo = 0;

        private readonly int linesPerPacket; 

        public RtpWriter(UdpClient udpClient, int width, int height, int rowPitch)
        {
            this.udpClient = udpClient;
            
            frameWidth = width;
            frameHeight = height;
            frameRowPitch = rowPitch;
            
            rtpHeader = new byte[12];
            rtpHeader[0] = 0x80;
            rtpHeader[1] = 0x7f;
            rtpHeader[8] = (byte) (ssrc >> 24 & 0xff);
            rtpHeader[9] = (byte) (ssrc >> 16 & 0xff);
            rtpHeader[10] = (byte) (ssrc >> 8 & 0xff);
            rtpHeader[11] = (byte) (ssrc & 0xff);

            linesPerPacket = Math.Max(1, (Mtu - RtpHeaderSize - Rfc4175HeaderSize) / frameWidth / Bpp);
            payloadHeader = new byte[Rfc4175HeaderSize + Rfc4175SegmentHeaderSize * linesPerPacket];
            payloadHeader[0] = (byte) (extSeqNo >> 8 & 0xff);
            payloadHeader[1] = (byte) (extSeqNo & 0xff);
        }
        
        public void WriteFrame(IntPtr buffer)
        {
            timestamp = (int)new DateTimeOffset(DateTime.UtcNow).ToUnixTimeMilliseconds();
            rtpHeader[4] = (byte) (timestamp >> 24 & 0xff);
            rtpHeader[5] = (byte) (timestamp >> 16 & 0xff);
            rtpHeader[6] = (byte) (timestamp >> 8 & 0xff);
            rtpHeader[7] = (byte) (timestamp & 0xff);
            
            var frameBytes = frameWidth * frameHeight * Bpp;
            var maxPayloadBytes = Mtu - RtpHeaderSize - Rfc4175HeaderSize - Rfc4175SegmentHeaderSize * linesPerPacket; 
            var bytesLeft = frameBytes;
            var lineNo = 0;
            var offsetPixels = 0;
            while (bytesLeft > 0)
            {
                sendBuffer.Seek(0, SeekOrigin.Begin);
                
                // write RTP header
                rtpHeader[2] = (byte) (seqNo >> 8 & 0xff);
                rtpHeader[3] = (byte) (seqNo & 0xff);
                seqNo++;

                var lineSegments = new (int bytes, bool fragmented)[linesPerPacket];
                for (var i = 0; i < linesPerPacket; i++)
                {
                    // write RFC4175 header
                    var linePixelsLeft = frameWidth - offsetPixels;
                    var lineBytesLeft = linePixelsLeft * Bpp;
                    var lineFragmented = lineBytesLeft > maxPayloadBytes;
                    var lineSegmentPixels = lineFragmented ? maxPayloadBytes / Bpp : linePixelsLeft;
                    var lineSegmentBytes = lineFragmented ? maxPayloadBytes : linePixelsLeft * Bpp;
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 0] = (byte) (lineSegmentBytes >> 8 & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 1] = (byte) (lineSegmentBytes & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 2] = (byte) (lineNo >> 8 & 0x7f);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 3] = (byte) (lineNo & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 4] = (byte) (offsetPixels >> 8 & 0x7f);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 5] = (byte) (offsetPixels & 0xff);
                    var continuation = i < linesPerPacket - 1;
                    if (continuation)
                    {
                        payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 4] |= 0x80; // set continuation bit
                    }
                    Console.WriteLine($"lineNo: {lineNo}, segmentBytes: {lineSegmentBytes}, pixel range: {offsetPixels}-{offsetPixels+lineSegmentPixels} (+{lineSegmentPixels} pixels)");
                    if (lineFragmented)
                    {
                        offsetPixels += lineSegmentPixels;
                    }
                    else
                    {
                        lineNo++;
                        offsetPixels = 0;
                    }
                    bytesLeft -= lineSegmentBytes;
                    lineSegments[i] = (bytes: lineSegmentBytes, fragmented: lineFragmented);
                    if (lineNo > frameHeight)
                    {
                        break;
                    }
                }
                Console.WriteLine("------ packet end");
                if (bytesLeft <= 0)
                {
                    rtpHeader[1] |= 0x80; // set marker bit
                }
                sendBuffer.Write(rtpHeader, 0, RtpHeaderSize);
                sendBuffer.Write(payloadHeader, 0, payloadHeader.Length);
                
                // write video frame payload
                foreach (var (bytes, fragmented) in lineSegments)
                {
                    unsafe
                    {
                        var lineSegmentPayload = new ReadOnlySpan<byte>(buffer.ToPointer(), bytes);
                        sendBuffer.Write(lineSegmentPayload);
                    }
                    buffer += bytes;
                    var hasEndOfLine = !fragmented;
                    if (hasEndOfLine)
                    {
                        buffer += frameRowPitch - frameWidth * Bpp;
                    }
                }
                
                udpClient.Send(sendBuffer.ToArray(), (int) sendBuffer.Length);
            }
            rtpHeader[1] &= 0x7f; // reset marker bit
        }
    }
}