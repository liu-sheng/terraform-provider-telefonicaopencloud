package telefonicaopencloud

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/elb/healthcheck"
)

const nameELBHC = "ELB-HealthCheck"

func resourceELBHealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceELBHealthCheckCreate,
		Read:   resourceELBHealthCheckRead,
		Update: resourceELBHealthCheckUpdate,
		Delete: resourceELBHealthCheckDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"listener_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"healthcheck_protocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					switch value {
					case "HTTP":
					case "TCP":
					default:
						errors = append(errors, fmt.Errorf("The valid value of %s is: HTTP, TCP", k))
					}
					return
				},
			},

			"healthcheck_uri": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					vv := regexp.MustCompile("^/[a-zA-Z0-9-/.%?#&_=]{0,79}$")
					if !vv.MatchString(value) {
						errors = append(errors, fmt.Errorf("%s is a string of 1 to 80 characters that must start with a slash (/) and can only contain letters, digits, and special characters such as -/.%?#&_=", k))
					}
					return
				},
			},

			"healthcheck_connect_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 || value > 65535 {
						errors = append(errors, fmt.Errorf("The value of %s must be in [1, 65535]", k))
					}
					return
				},
			},

			"healthy_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 || value > 10 {
						errors = append(errors, fmt.Errorf("The value of %s must be in [1, 10]", k))
					}
					return
				},
			},

			"unhealthy_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 || value > 10 {
						errors = append(errors, fmt.Errorf("The value of %s must be in [1, 10]", k))
					}
					return
				},
			},

			"healthcheck_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 || value > 50 {
						errors = append(errors, fmt.Errorf("The value of %s must be in [1, 50]", k))
					}
					return
				},
			},

			"healthcheck_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 || value > 5 {
						errors = append(errors, fmt.Errorf("The value of %s must be in [1, 5]", k))
					}
					return
				},
			},

			"update_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"create_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceELBHealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := chooseELBClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var createOpts healthcheck.CreateOpts
	err = buildELBCreateParam(&createOpts, d)
	if err != nil {
		return fmt.Errorf("Error creating %s: building parameter failed:%s", nameELBHC, err)
	}
	log.Printf("[DEBUG] Create %s Options: %#v", nameELBHC, createOpts)

	hc, err := healthcheck.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating %s: %s", nameELBHC, err)
	}
	log.Printf("[DEBUG] Create %s: %#v", nameELBHC, *hc)

	// If all has been successful, set the ID on the resource
	d.SetId(hc.ID)

	return resourceELBHealthCheckRead(d, meta)
}

func resourceELBHealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := chooseELBClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	hc, err := healthcheck.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "healthcheck")
	}
	log.Printf("[DEBUG] Retrieved %s %s: %#v", nameELBHC, d.Id(), hc)

	return refreshResourceData(hc, d)
}

func resourceELBHealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := chooseELBClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	hcId := d.Id()

	var updateOpts healthcheck.UpdateOpts
	err = buildELBUpdateParam(&updateOpts, d)
	if err != nil {
		return fmt.Errorf("Error updating %s(%s): building parameter failed:%s", nameELBHC, hcId, err)
	}
	b, err := updateOpts.IsNeedUpdate()
	if err != nil {
		return err
	}
	if !b {
		log.Printf("[INFO] Updating %s %s with no changes", nameELBHC, hcId)
		return nil
	}
	log.Printf("[DEBUG] Updating healthcheck %s(%s) with options: %#v", nameELBHC, hcId, updateOpts)

	timeout := d.Timeout(schema.TimeoutUpdate)
	err = resource.Retry(timeout, func() *resource.RetryError {
		_, err := healthcheck.Update(networkingClient, hcId, updateOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error updating %s(%s): %s", nameELBHC, hcId, err)
	}

	return resourceELBHealthCheckRead(d, meta)
}

func resourceELBHealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := chooseELBClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	hcId := d.Id()
	log.Printf("[DEBUG] Deleting %s %s", nameELBHC, hcId)

	timeout := d.Timeout(schema.TimeoutDelete)
	err = resource.Retry(timeout, func() *resource.RetryError {
		err := healthcheck.Delete(networkingClient, hcId).ExtractErr()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})
	if err != nil {
		if isELBResourceNotFound(err) {
			log.Printf("[INFO] deleting an unavailable %s: %s", nameELBHC, hcId)
			return nil
		}
		return fmt.Errorf("Error deleting %s(%s): %s", nameELBHC, hcId, err)
	}

	return nil
}
