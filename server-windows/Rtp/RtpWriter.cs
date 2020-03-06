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
            
        private const short Mtu = 1500;
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
            
            var frameSize = frameWidth * frameHeight * Bpp;
            var maxPayloadSize = Mtu - RtpHeaderSize - Rfc4175HeaderSize - Rfc4175SegmentHeaderSize * linesPerPacket; 
            var bytesLeft = frameSize;
            var lineNo = 0;
            var pixelOffset = 0;
            while (bytesLeft > 0)
            {
                sendBuffer.Seek(0, SeekOrigin.Begin);
                
                // write RTP header
                rtpHeader[2] = (byte) (seqNo >> 8 & 0xff);
                rtpHeader[3] = (byte) (seqNo & 0xff);
                seqNo++;

                // write RFC4175 header
                var lineSize = (frameWidth - pixelOffset) * Bpp;
                var addOffset = 0;
                var isPartialLine = lineSize > maxPayloadSize;
                if (isPartialLine)
                {
                    lineSize = maxPayloadSize; // offset bytes
                    addOffset = lineSize / Bpp; // offset pixels
                }
                else
                {
                    pixelOffset = 0;
                }
                for (var i = 0; i < linesPerPacket; i++)
                {
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 0] = (byte) (lineSize >> 8 & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 1] = (byte) (lineSize & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 2] = (byte) (lineNo >> 8 & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 3] = (byte) (lineNo & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 4] = (byte) (pixelOffset >> 8 & 0xff);
                    payloadHeader[2 + i * Rfc4175SegmentHeaderSize + 5] = (byte) (pixelOffset & 0xff);
                    var continuation = i < linesPerPacket - 1;
                    if (continuation)
                    {
                        payloadHeader[i * Rfc4175SegmentHeaderSize + 4] |= 0x80; // set continuation bit
                    }
                }
                if (isPartialLine)
                {
                    pixelOffset += addOffset;
                }
                else
                {
                    lineNo++;
                }
                
                var payloadSize = lineSize * linesPerPacket;
                bytesLeft -= payloadSize;
                Console.WriteLine("bytesLeft: {0}", bytesLeft);
                if (bytesLeft <= 0)
                {
                    rtpHeader[1] |= 0x80; // set marker bit
                }
                sendBuffer.Write(rtpHeader, 0, RtpHeaderSize);
                sendBuffer.Write(payloadHeader, 0, payloadHeader.Length);
                
                // write video frame payload
                for (var i = 0; i < linesPerPacket; i++)
                {
                    unsafe
                    {
                        var linePayload = new ReadOnlySpan<byte>(buffer.ToPointer(), lineSize);
                        sendBuffer.Write(linePayload);
                    }
                    buffer += lineSize;
                    if (!isPartialLine)
                    {
                        // adjust rowPitch line
                        buffer += frameRowPitch - (pixelOffset * Bpp + lineSize);
                    }
                }
                
                udpClient.Send(sendBuffer.ToArray(), (int) sendBuffer.Length);
            }
            rtpHeader[1] &= 0x7f; // reset marker bit
        }
    }
}