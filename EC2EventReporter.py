import boto3
import json
import datetime
import time

#Create variables
avgHoursPerMonth = ((365.25 / 12) * 24)
regionName = "us-west-2"

#Create clients
ec2Client = boto3.client('ec2', region_name=regionName)
cloudTrailClient = boto3.client('cloudtrail', region_name=regionName)
sesClient = boto3.client('ses', region_name=regionName)
ec2Resource = boto3.resource('ec2')





#Create instances Dict that contains instances types and their corresponding price
#format:
#'type':        price,
instances = {

't2.nano':      0.0059,
't2.micro':     0.012,
't2.small':     0.023,
't2.medium':    0.047,
't2.large':     0.094,
't2.xlarge':    0.188,
't2.2xlarge':   0.376,

'm4.large':     0.108,
'm4.xlarge':    0.215,
'm4.2xlarge':   0.431,
'm4.4xlarge':   0.862,
'm4.1large':    2.155,
'm4.xlarge':    3.447,

'm3.medium':    0.067,
'm3.large':     0.133,
'm3.xlarge':    0.266,
'm3.2xlarge':   0.532,

'c4.large':     0.1,
'c4.xlarge':    0.199,
'c4.2xlarge':   0.398,
'c4.4xlarge':   0.796,
'c4.8xlarge':   1.591,

'c3.large':     0.105,
'c3.xlarge':    0.21,
'c3.2xlarge':   0.42,
'c3.4xlarge':   0.84,
'c3.8xlarge':   1.68,

'p2.xlarge':    0.9,
'p2.8xlarge':   7.2,
'p2.xlarge':    14.4,

'g2.2xlarge':   0.65,
'g2.8xlarge':   2.6,

'x1.xlarge':    6.669,
'x1.xlarge':    13.338,

'r3.large':     0.166,
'r3.xlarge':    0.333,
'r3.2xlarge':   0.665,
'r3.4xlarge':   1.33,
'r3.8xlarge':   2.66,

'r4.large':     0.133,
'r4.xlarge':    0.266,
'r4.2xlarge':   0.532,
'r4.4xlarge':   1.064,
'r4.8xlarge':   2.128,
'r4.xlarge':    4.256,

'i2.xlarge':    0.853,
'i2.2xlarge':   1.705,
'i2.4xlarge':   3.41,
'i2.8xlarge':   6.82,

'd2.xlarge':    0.69,
'd2.2xlarge':   1.38,
'd2.4xlarge':   2.76,
'd2.8xlarge':   5.52,

't1.micro':     0.02,

}

volumes = {

'standard':     .05,
'gp2':          0.1,
'io1':          { 'GB' : 0.125, 'IOPS' : .065 },
'st1':          0.045,
'sc1':          0.025,

}


#create exceptions
class ExceptionArryTooLong(Exception):
    pass


class ExceptionArryTooShort(Exception):
    pass


#Turn arrays with one item into variables with the value of that item
def flattenArray(array):
    if len(array) > 1:
        raise ExceptionArryTooLong()

    elif len(array) < 1:
        raise ExceptionArryTooShort()

    for item in array:
        newValue = item

    return newValue



def lambda_handler(event, context):
    totalPrice = 0
    avgBlockCost = 0


    #Get Instance ID
    instanceID = event.get('detail').get('responseElements').get('instancesSet').get('items')[0].get('instanceId')
    userName = event.get('detail').get('userIdentity').get('userName')
    timeOfInstance = event.get('detail').get('userIdentity').get('sessionContext').get('attributes').get('creationDate')


    if (event.get('detail').get('eventName') == "RunInstances"):
        instanceType = event.get('detail').get('requestParameters').get('instanceType')


    else:
        #Get Instance Type
        instanceTypeAttr = ec2Client.describe_instance_attribute(InstanceId = instanceID, Attribute = 'instanceType')
        instanceType = flattenArray(instanceTypeAttr.get("InstanceType").values())


    #Get Volume Type, Size, and iops
    instanceBlockAttr = ec2Client.describe_instance_attribute(InstanceId = instanceID, Attribute = 'blockDeviceMapping')
    blockVolumeID = flattenArray(instanceBlockAttr.get("BlockDeviceMappings")).get("Ebs").get("VolumeId")
    blockVolumeIDList = [blockVolumeID]


    volumeDescription = ec2Client.describe_volumes(DryRun = False, VolumeIds = blockVolumeIDList)
    volumeType =  flattenArray(volumeDescription.get("Volumes")).get("VolumeType")
    blockIops = flattenArray(volumeDescription.get("Volumes")).get("Iops")
    blockSize = flattenArray(volumeDescription.get("Volumes")).get("Size")



    #Get Instance Name
    instanceIDList = []
    instanceIDList.append(instanceID)
    instanceDescription = ec2Client.describe_instances(InstanceIds = instanceIDList)
    del instanceIDList[:]

    try:
        reservations = flattenArray(instanceDescription.get('Reservations'))
        instance = flattenArray(reservations.get('Instances'))
        tags = flattenArray(instance.get('Tags'))
        instanceName = tags.get('Value')

    except ExceptionArryTooLong:
        instanceName = "not found"

    except ExceptionArryTooShort:
        instanceName = "not found"





    if volumeType in volumes:
        #Calculates price based on gb used per month
        if volumeType != "io1":
            avgBlockCost = round(volumes.get(volumeType) * blockSize, 2)
            avgBlockCostStr = "Average Monthly Volume Cost: $" + str(avgBlockCost)
            totalPrice = avgBlockCost

        #Calculates price based on iops and gb used per month
        else:
            avgBlockCost = round(volumes.get(volumeType).get("GB") * blockSize) + (volumes.get(volumeType).get("IOPS") * blockIops, 2)
            avgBlockCostStr = "Average Monthly Volume Cost: $" + str(avgBlockCost)
            totalPrice = avgBlockCost

    else:
        avgBlockCostStr = ( "Average Monthly Volume Cost: Not Found" + "\n\n" +
        "The volume price is not available. Go to https://aws.amazon.com/ebs/pricing/ for volume price. " +
        "Go to https://console.aws.amazon.com/lambda/home?region=us-east-1#/functions/EC2EventReporter?tab=code" +
        " to edit the dictionary.")





    #Check if the instance type found is in the instances dict
    if instanceType in instances:
        #Average monthly cost
        avgInstanceCost = round(instances[instanceType] * avgHoursPerMonth, 2)

        #create a variable that will be the contents of the email
        #if instance type has a price in the dict
        avgInstanceCostStr = ("Average Monthly Instance Price: $" + str(avgInstanceCost))
        totalPrice += avgInstanceCost


    else:
        #create a variable that will be the contents of the email
        #if instance type does not have a price in the dict
        avgInstanceCostStr = (
        "Average Monthly Instance Price: Not Found" + "\n\n" +
        "The instance price is not available. Go to https://aws.amazon.com/ec2/pricing/on-demand/ for instance price." +
        "Go to https://us-west-2.console.aws.amazon.com/lambda/home?region=us-west-2#/functions/EC2InstancePriceCheck?tab=code" +
        " to edit the dictionary."
        )


    #if the total price is the same as either the instance price or the block price
    #don't add the total cost to the email
    totalPriceStr = "Total Price: $" + str(totalPrice)

    emailBody = (
        "Username: " + userName + "\n" +
        "Instance Name: " + instanceName + "\n" +
        "Instance ID: " + instanceID + "\n" +
        "Instance Type: " + instanceType + "\n" +
        avgInstanceCostStr + "\n\n" +
        "Volume Type: " + volumeType + "\n" +
        "Provisioned IOPS: " + str(blockIops) + "\n" +
        "Provisioned Storage: " + str(blockSize) + "GB \n" +
        avgBlockCostStr + "\n\n" +
        totalPriceStr

    )





    # Send Email
    email = sesClient.send_email(
        Source = 'sysops@riskanalytics.com',
        Destination={
            'ToAddresses' : [
                 'swerremeyer@riskanalytics.com', #'bcrook@riskanalytics.com',
            ],
        },
        Message= {
            'Subject' : {
                'Data' : 'New EC2 Instance',
                'Charset' : 'utf8'
            },
            'Body' : {
                'Text' : {
                    'Data' : emailBody,
                    'Charset' : 'utf8'
                }
            }
        }
    )
